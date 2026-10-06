package tui

func (f *spawnForm) blueprintArgs() []string {
	for _, b := range f.blueprints {
		if b.Name == f.opts.Blueprint {
			return b.Args
		}
	}
	return nil
}
