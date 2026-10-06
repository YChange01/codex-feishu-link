package control

import "testing"

func TestModelPageOffersModelAndEffortInOneForm(t *testing.T) {
	page := modelPageViewFromCommandConfigView(FeishuCatalogConfigView{
		CommandID:            FeishuCommandModel,
		FormOptions:          []CommandCatalogFormFieldOption{{Label: "Model A", Value: "model-a"}},
		SecondaryFormOptions: []CommandCatalogFormFieldOption{{Label: "自动", Value: "clear"}, {Label: "high", Value: "high"}},
		OverrideValue:        "model-a",
		OverrideExtraValue:   "high",
	})
	if len(page.Sections) == 0 || len(page.Sections[0].Entries) == 0 {
		t.Fatalf("missing combined model section: %#v", page)
	}
	form := page.Sections[0].Entries[0].Form
	if form == nil || form.Field.Name != modelPresetCommandFieldName || form.SecondaryField == nil || form.SecondaryField.Name != "command_args_effort" {
		t.Fatalf("model and effort are not in one form: %#v", form)
	}
	if form.Field.DefaultValue != "model-a" || form.SecondaryField.DefaultValue != "high" {
		t.Fatalf("combined form lost current selection: %#v", form)
	}
	if form.SecondaryField.Options[0].Value != "clear" {
		t.Fatalf("missing automatic effort: %#v", form.SecondaryField.Options)
	}
}
