package claude

import (
	"testing"
	"time"
)

func TestActivityTracksConcurrentSubagents(t *testing.T) {
	var a Activity
	stamp := time.Now().UTC()
	steps := []struct{ payload, want string }{
		{`{"hook_event_name":"UserPromptSubmit","session_id":"parent"}`, "working"},
		{`{"hook_event_name":"SubagentStart","session_id":"parent","agent_id":"one"}`, "working"},
		{`{"hook_event_name":"SubagentStart","session_id":"parent","agent_id":"two"}`, "working"},
		{`{"hook_event_name":"SubagentStart","session_id":"parent","agent_id":"two"}`, "working"},
		{`{"hook_event_name":"Stop","session_id":"parent","last_assistant_message":"Waiting for reviews"}`, "working"},
		{`{"hook_event_name":"Notification","session_id":"parent","notification_type":"idle_prompt"}`, "working"},
		{`{"hook_event_name":"SubagentStop","session_id":"parent","agent_id":"one"}`, "working"},
		{`{"hook_event_name":"SubagentStop","session_id":"parent","agent_id":"one"}`, "working"},
		{`{"hook_event_name":"PreToolUse","session_id":"parent","tool_name":"AskUserQuestion","tool_input":{"questions":[{"question":"Choose a plan"}]}}`, "question"},
		{`{"hook_event_name":"PostToolUse","session_id":"parent","agent_id":"two","tool_name":"Read"}`, "question"},
		{`{"hook_event_name":"PostToolUse","session_id":"parent","tool_name":"AskUserQuestion"}`, "working"},
		{`{"hook_event_name":"Stop","session_id":"parent","last_assistant_message":"Done"}`, "working"},
		{`{"hook_event_name":"SubagentStop","session_id":"parent","agent_id":"two"}`, "ready"},
		{`{"hook_event_name":"SubagentStart","session_id":"parent","agent_id":"three"}`, "working"},
		{`{"hook_event_name":"SessionEnd","session_id":"parent"}`, "ended"},
		{`{"hook_event_name":"SessionStart","session_id":"next"}`, "idle"},
	}
	for i, step := range steps {
		e, err := Parse([]byte(step.payload), "")
		if err != nil {
			t.Fatal(err)
		}
		e.TS = stamp.Add(time.Duration(i) * time.Second)
		result := a.Apply(e)
		if result.Status != step.want {
			t.Fatalf("step %d: got %+v, want %s", i, result, step.want)
		}
		if result.Status != "ended" {
			live := "idle"
			if len(a.Children) == 0 && result.Status == "working" {
				live = "working"
			}
			if status, _ := a.Status(e.AgentSessionID, live, stamp.Add(-time.Second).UnixMilli()); status != step.want {
				t.Fatalf("discovery erased activity at step %d: %s", i, status)
			}
		}
	}
	if len(a.Children) != 0 {
		t.Fatal("children survived new session")
	}
	if status, since := a.Status("other", "idle", 0); status != "idle" || since != 0 {
		t.Fatal("old session evidence reused")
	}
	if status, since := a.Status("next", "busy", time.Now().Add(time.Hour).UnixMilli()); status != "busy" || since != 0 {
		t.Fatal("old process evidence reused")
	}
}

func TestStopBackgroundSnapshotRecoversMissedSubagentHooks(t *testing.T) {
	var a Activity
	for _, step := range []struct{ payload, want string }{
		{`{"hook_event_name":"Stop","session_id":"parent","background_tasks":[{"id":"review","type":"subagent","status":"running"}]}`, "working"},
		{`{"hook_event_name":"SubagentStop","session_id":"parent","agent_id":"review","background_tasks":[]}`, "ready"},
	} {
		e, err := Parse([]byte(step.payload), "")
		if err != nil {
			t.Fatal(err)
		}
		e.TS = time.Now()
		if got := a.Apply(e); got.Status != step.want {
			t.Fatal(got)
		}
	}
}

func TestBusyBackgroundShellAfterFinishedTurnIsWaiting(t *testing.T) {
	stamp := time.Now().UTC()
	started := stamp.Add(-time.Minute).UnixMilli()
	apply := func(a *Activity, payload string) {
		t.Helper()
		e, err := Parse([]byte(payload), "")
		if err != nil {
			t.Fatal(err)
		}
		e.TS = stamp
		a.Apply(e)
	}
	var a Activity
	apply(&a, `{"hook_event_name":"Stop","session_id":"s","background_tasks":[{"id":"e2e","type":"shell","status":"running"}]}`)
	if status, _ := a.Status("s", "working", started); status != "ready" {
		t.Fatalf("finished turn with a background shell: %s", status)
	}
	apply(&a, `{"hook_event_name":"Notification","session_id":"s","notification_type":"idle_prompt"}`)
	if status, _ := a.Status("s", "working", started); status != "idle" {
		t.Fatalf("idle prompt with a background shell: %s", status)
	}
	var sub Activity
	apply(&sub, `{"hook_event_name":"Stop","session_id":"s","background_tasks":[{"id":"review","type":"subagent","status":"running"}]}`)
	if status, _ := sub.Status("s", "working", started); status != "working" {
		t.Fatalf("running subagent: %s", status)
	}
	var prompted Activity
	apply(&prompted, `{"hook_event_name":"UserPromptSubmit","session_id":"s","prompt":"go"}`)
	if status, _ := prompted.Status("s", "working", started); status != "working" {
		t.Fatalf("new prompt: %s", status)
	}
	if status, _ := (Activity{}).Status("s", "working", started); status != "working" {
		t.Fatalf("no hook evidence: %s", status)
	}
}
