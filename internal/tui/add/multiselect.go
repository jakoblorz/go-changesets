package add

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	huh "github.com/charmbracelet/huh"
)

type projectMultiSelect struct {
	*huh.MultiSelect[string]
	selected *[]string
	keymap   *huh.KeyMap
}

func newProjectMultiSelect(selected *[]string) *projectMultiSelect {
	return &projectMultiSelect{
		MultiSelect: huh.NewMultiSelect[string]().Value(selected),
		selected:    selected,
	}
}

func (p *projectMultiSelect) Options(options ...huh.Option[string]) *projectMultiSelect {
	p.MultiSelect.Options(options...)
	return p
}

func (p *projectMultiSelect) WithKeyMap(k *huh.KeyMap) huh.Field {
	p.keymap = k
	p.MultiSelect.WithKeyMap(k)
	return p
}

func (p *projectMultiSelect) KeyBinds() []key.Binding {
	binds := p.MultiSelect.KeyBinds()
	if p.keymap == nil {
		return binds
	}

	helpDesc := "continue"
	if p.selectedCount() == 0 {
		helpDesc = "select and continue"
	}

	submitKeys := p.keymap.MultiSelect.Submit.Keys()
	if len(submitKeys) == 0 {
		return binds
	}

	for i := range binds {
		if !bindingHasKeys(binds[i], submitKeys) {
			continue
		}
		helpKey := binds[i].Help().Key
		if helpKey == "" {
			helpKey = submitKeys[0]
		}
		if helpKey == "" {
			helpKey = "enter"
		}
		binds[i].SetHelp(helpKey, helpDesc)
		break
	}

	return binds
}

func (p *projectMultiSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if p.keymap != nil {
		if keyMsg, ok := msg.(tea.KeyMsg); ok && key.Matches(keyMsg, p.keymap.MultiSelect.Submit) {
			if p.selectedCount() == 0 {
				if _, ok := p.MultiSelect.Hovered(); ok {
					toggleMsg, ok := keyMsgForBinding(p.keymap.MultiSelect.Toggle)
					if !ok {
						toggleMsg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
					}
					model, cmd := p.MultiSelect.Update(toggleMsg)
					p.MultiSelect = model.(*huh.MultiSelect[string])
					cmds = append(cmds, cmd)
				}
			}
		}
	}

	model, cmd := p.MultiSelect.Update(msg)
	p.MultiSelect = model.(*huh.MultiSelect[string])
	cmds = append(cmds, cmd)
	return p, tea.Batch(cmds...)
}

func keyMsgForBinding(binding key.Binding) (tea.KeyMsg, bool) {
	for _, label := range binding.Keys() {
		if msg, ok := keyMsgFromLabel(label); ok {
			return msg, true
		}
	}
	return tea.KeyMsg{}, false
}

func keyMsgFromLabel(label string) (tea.KeyMsg, bool) {
	switch label {
	case " ", "space":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}, true
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}, true
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}, true
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}, true
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}, true
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}, true
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}, true
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}, true
	case "esc", "escape":
		return tea.KeyMsg{Type: tea.KeyEsc}, true
	}

	runes := []rune(label)
	if len(runes) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: runes}, true
	}

	return tea.KeyMsg{}, false
}

func (p *projectMultiSelect) selectedCount() int {
	value, ok := p.MultiSelect.GetValue().([]string)
	if !ok {
		return 0
	}
	return len(value)
}

func bindingHasKeys(binding key.Binding, keys []string) bool {
	bindingKeys := binding.Keys()
	if len(bindingKeys) != len(keys) {
		return false
	}
	for i := range keys {
		if bindingKeys[i] != keys[i] {
			return false
		}
	}
	return true
}
