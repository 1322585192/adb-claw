package observe

// UIDelta is a compact difference between two UI trees of the same page.
type UIDelta struct {
	Added   []Element `json:"added,omitempty"`
	Removed []string  `json:"removed,omitempty"`
	Changed []Element `json:"changed,omitempty"`
	Same    bool      `json:"same,omitempty"`
}

// DiffTrees returns added/removed/changed interactive nodes.
func DiffTrees(prev, curr *UITree) *UIDelta {
	if curr == nil {
		return &UIDelta{}
	}
	if prev == nil {
		return &UIDelta{Added: curr.Elements}
	}
	if prev.Hash() == curr.Hash() {
		return &UIDelta{Same: true}
	}

	prevByKey := map[string]Element{}
	for _, el := range prev.Elements {
		prevByKey[elementKey(el)] = el
	}
	currByKey := map[string]Element{}
	for _, el := range curr.Elements {
		currByKey[elementKey(el)] = el
	}

	delta := &UIDelta{}
	for key, el := range currByKey {
		if old, ok := prevByKey[key]; !ok {
			delta.Added = append(delta.Added, el)
		} else if old.Center != el.Center || old.Clickable != el.Clickable || old.Enabled != el.Enabled {
			delta.Changed = append(delta.Changed, el)
		}
	}
	for key, el := range prevByKey {
		if _, ok := currByKey[key]; !ok {
			handle := el.Handle
			if handle == "" {
				handle = elementKey(el)
			}
			delta.Removed = append(delta.Removed, handle)
		}
	}
	return delta
}

func elementKey(el Element) string {
	if el.ResourceID != "" {
		return "id:" + el.ResourceID
	}
	if el.Text != "" {
		return "t:" + el.Text
	}
	if el.ContentDesc != "" {
		return "d:" + el.ContentDesc
	}
	return el.Handle
}
