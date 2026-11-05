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

func TestBusyBackgroundShellAfterFinishedTurnIsVisible(t *testing.T) {
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
	if status, _ := a.Status("s", "working", started); status != "background" {
		t.Fatalf("finished turn with a background shell: %s", status)
	}
	apply(&a, `{"hook_event_name":"Notification","session_id":"s","notification_type":"idle_prompt"}`)
	if status, _ := a.Status("s", "working", started); status != "background" {
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

func TestProcessActivityLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps []struct{ payload, status string }
	}{
		{"blocking shell", []struct{ payload, status string }{
			{`{"hook_event_name":"PreToolUse","session_id":"s","tool_name":"Bash","tool_input":{"command":"sleep 10"}}`, "waiting"},
			{`{"hook_event_name":"PostToolUse","session_id":"s","tool_name":"Bash"}`, "working"},
		}},
		{"failure", []struct{ payload, status string }{
			{`{"hook_event_name":"PreToolUse","session_id":"s","tool_name":"Bash"}`, "waiting"},
			{`{"hook_event_name":"PostToolUseFailure","session_id":"s","tool_name":"Bash"}`, "working"},
		}},
		{"permission", []struct{ payload, status string }{
			{`{"hook_event_name":"PreToolUse","session_id":"s","tool_name":"Bash"}`, "waiting"},
			{`{"hook_event_name":"Notification","session_id":"s","notification_type":"permission_prompt"}`, "permission"},
			{`{"hook_event_name":"PermissionDenied","session_id":"s","tool_name":"Bash"}`, "working"},
		}},
		{"monitor", []struct{ payload, status string }{
			{`{"hook_event_name":"Stop","session_id":"s","background_tasks":[{"id":"m","type":"monitor","status":"running"}]}`, "background"},
			{`{"hook_event_name":"Notification","session_id":"s","notification_type":"idle_prompt"}`, "background"},
			{`{"hook_event_name":"Notification","session_id":"s","notification_type":"permission_prompt"}`, "permission"},
			{`{"hook_event_name":"UserPromptSubmit","session_id":"s"}`, "working"},
			{`{"hook_event_name":"Stop","session_id":"s","background_tasks":[]}`, "ready"},
		}},
		{"restart", []struct{ payload, status string }{
			{`{"hook_event_name":"Stop","session_id":"s","background_tasks":[{"id":"p","type":"shell","status":"running"}]}`, "background"},
			{`{"hook_event_name":"SessionStart","session_id":"s"}`, "idle"},
		}},
		{"all children waiting", []struct{ payload, status string }{
			{`{"hook_event_name":"Stop","session_id":"s"}`, "ready"},
			{`{"hook_event_name":"PreToolUse","session_id":"s","agent_id":"a","tool_name":"Bash"}`, "waiting"},
			{`{"hook_event_name":"SubagentStart","session_id":"s","agent_id":"b"}`, "working"},
			{`{"hook_event_name":"SubagentStop","session_id":"s","agent_id":"b"}`, "waiting"},
			{`{"hook_event_name":"SubagentStop","session_id":"s","agent_id":"a"}`, "ready"},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var a Activity
			for i, step := range tc.steps {
				e, err := Parse([]byte(step.payload), "")
				if err != nil {
					t.Fatal(err)
				}
				e.TS = time.Now().Add(time.Duration(i) * time.Second)
				if got := a.Apply(e); got.Status != step.status {
					t.Fatalf("step %d: got %s, want %s", i, got.Status, step.status)
				}
			}
		})
	}
}

func TestProcessDiscoveryReconciliation(t *testing.T) {
	stamp := time.Now()
	for _, tc := range []struct {
		name, tasks, hook, live, want string
		summary                       bool
	}{
		{"shell busy", `[{"id":"s","type":"shell","status":"running"}]`, "", "working", "background", true},
		{"shell finished", `[{"id":"s","type":"shell","status":"running"}]`, "", "idle", "ready", false},
		{"monitor armed", `[{"id":"m","type":"monitor","status":"running"}]`, "", "idle", "background", true},
		{"empty snapshot", `[]`, "", "idle", "ready", false},
		{"completed task", `[{"id":"s","type":"shell","status":"completed"}]`, "", "idle", "ready", false},
		{"live permission", `[{"id":"s","type":"shell","status":"running"}]`, "", "permission", "permission", true},
		{"approved permission", `[{"id":"s","type":"shell","status":"running"}]`, `{"hook_event_name":"Notification","session_id":"s","notification_type":"permission_prompt"}`, "working", "working", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var a Activity
			e, err := Parse([]byte(`{"hook_event_name":"Stop","session_id":"s","background_tasks":`+tc.tasks+`}`), "")
			if err != nil {
				t.Fatal(err)
			}
			e.TS = stamp
			a.Apply(e)
			if tc.hook != "" {
				e, err = Parse([]byte(tc.hook), "")
				if err != nil {
					t.Fatal(err)
				}
				e.TS = stamp
				a.Apply(e)
			}
			if got, _ := a.Status("s", tc.live, stamp.Add(-time.Minute).UnixMilli()); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
			if summary := a.ProcessSummary("s", tc.live, 0); (summary != "") != tc.summary {
				t.Fatalf("summary: %q", summary)
			}
			if got, _ := a.Status("other", "idle", 0); got != "idle" {
				t.Fatal("reused other session")
			}
			if a.ProcessSummary("s", "working", stamp.Add(time.Minute).UnixMilli()) != "" {
				t.Fatal("reused previous process")
			}
		})
	}
}

func TestProcessCountsClearedByChildSnapshot(t *testing.T) {
	var a Activity
	for _, payload := range []string{
		`{"hook_event_name":"Stop","session_id":"s","background_tasks":[{"id":"p","type":"shell","status":"running"}]}`,
		`{"hook_event_name":"SubagentStop","session_id":"s","agent_id":"child","background_tasks":[]}`,
	} {
		e, err := Parse([]byte(payload), "")
		if err != nil {
			t.Fatal(err)
		}
		e.TS = time.Now()
		a.Apply(e)
	}
	if a.Current.Status != "ready" || a.Current.Detail["shells"] != "" || a.Current.Detail["monitors"] != "" {
		t.Fatalf("stale process display: %+v", a.Current)
	}
}
