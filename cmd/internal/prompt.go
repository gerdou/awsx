package internal

import (
	"fmt"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/manifoldco/promptui"
	"strings"
)

type Prompt interface {
	Select(label string, toSelect []string, searcher func(input string, index int) bool) (index int, value string, err error)
	MultiSelect(label string, toSelect []string, searcher func(input string, index int) bool) ([]int, error)
	Prompt(label string, dfault string) (string, error)
}

type Prompter struct{}

type multiSelectStruct struct {
	ID         int
	IsSelected bool
	Label      string
}

func (receiver Prompter) Select(label string, toSelect []string, searcher func(input string, index int) bool) (int, string, error) {
	templates := &promptui.SelectTemplates{
		Label:    fmt.Sprintf("%s {{.}}: ", promptui.IconInitial),
		Active:   fmt.Sprintf("%s {{ . | underline }}", promptui.IconSelect),
		Inactive: "  {{.}}",
		Selected: fmt.Sprintf(`{{ "%s" | green }} {{ . | faint }}`, promptui.IconGood),
	}
	prompt := promptui.Select{
		Label:             label,
		Items:             toSelect,
		Size:              20,
		Searcher:          searcher,
		StartInSearchMode: searcher != nil,
		Templates:         templates,
	}
	index, value, err := prompt.Run()
	if err != nil {
		return 0, "", err
	}
	return index, value, nil
}

func (receiver Prompter) MultiSelect(label string, toSelect []string, searcher func(input string, index int) bool) ([]int, error) {
	if len(toSelect) == 0 {
		return []int{}, nil
	}

	if len(toSelect) == 1 {
		return []int{0}, nil
	}

	templates := &promptui.SelectTemplates{
		Label:    fmt.Sprintf("%s {{.}}: ", promptui.IconInitial),
		Active:   fmt.Sprintf("%s {{if .IsSelected}}%s {{end}}{{ .Label | underline }}", promptui.IconSelect, promptui.IconGood),
		Inactive: fmt.Sprintf("  {{if .IsSelected}}%s {{end}}{{ .Label }}", promptui.IconGood),
	}

	toSelectStructs := make([]*multiSelectStruct, 0, len(toSelect)+1)
	doneLabel := "Done"

	toSelectStructs = append(toSelectStructs, &multiSelectStruct{
		ID:         0,
		IsSelected: false,
		Label:      doneLabel,
	})

	for index, item := range toSelect {
		toSelectStructs = append(toSelectStructs, &multiSelectStruct{
			ID:         index + 1,
			IsSelected: false,
			Label:      item,
		})
	}

	selected, err := multiSelect(label, toSelectStructs, searcher, templates, 1, make(map[int]struct{}))

	result := make([]int, 0, len(selected))
	for index := range selected {
		result = append(result, index-1)
	}

	return result, err
}

func multiSelect(label string, toSelect []*multiSelectStruct, searcher func(input string, index int) bool, templates *promptui.SelectTemplates, defaultIndex int, selectedSoFar map[int]struct{}) (map[int]struct{}, error) {
	if selectedSoFar == nil {
		selectedSoFar = make(map[int]struct{})
	}

	prompt := promptui.Select{
		Label:             label,
		Items:             toSelect,
		Size:              20,
		Searcher:          searcher,
		StartInSearchMode: searcher != nil,
		Templates:         templates,
		CursorPos:         defaultIndex,
		HideSelected:      true,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return selectedSoFar, err
	}

	if index == 0 {
		// Done selected
		return selectedSoFar, nil
	}

	toSelect[index].IsSelected = !toSelect[index].IsSelected

	switch toSelect[index].IsSelected {
	case true:
		selectedSoFar[index] = struct{}{}
	case false:
		if _, exists := selectedSoFar[index]; exists {
			delete(selectedSoFar, index)
		}
	}

	return multiSelect(label, toSelect, searcher, templates, index, selectedSoFar)
}

func (receiver Prompter) Prompt(label string, dfault string) (string, error) {
	prompt := promptui.Prompt{
		Label:     label,
		Default:   dfault,
		AllowEdit: false,
	}
	val, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return val, nil
}

func fuzzySearchWithPrefixAnchor(itemsToSelect []string, linePrefix string) func(input string, index int) bool {
	return func(input string, index int) bool {
		role := itemsToSelect[index]

		if strings.HasPrefix(input, linePrefix) {
			if strings.HasPrefix(role, input) {
				return true
			}
			return false
		} else {
			if fuzzy.MatchFold(input, role) {
				return true
			}
		}
		return false
	}
}
