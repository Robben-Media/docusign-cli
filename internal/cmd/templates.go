package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/builtbyrobben/docusign-cli/internal/outfmt"
)

type TemplatesCmd struct {
	List TemplatesListCmd `cmd:"" help:"List templates"`
	Get  TemplatesGetCmd  `cmd:"" help:"Get template details"`
}

type TemplatesListCmd struct {
	Search string `help:"Search text to filter templates"`
	Count  int    `help:"Number of results to return" default:"10"`
}

func (cmd *TemplatesListCmd) Run(ctx context.Context) error {
	client, err := getDocuSignClient(ctx)
	if err != nil {
		return err
	}

	result, err := client.ListTemplates(ctx, cmd.Search, cmd.Count)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}
	if outfmt.IsPlain(ctx) {
		headers := []string{"ID", "NAME", "DESCRIPTION", "MODIFIED"}
		var rows [][]string
		for _, tmpl := range result.EnvelopeTemplates {
			rows = append(rows, []string{tmpl.TemplateID, tmpl.Name, tmpl.Description, tmpl.LastModified})
		}
		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	if len(result.EnvelopeTemplates) == 0 {
		fmt.Fprintln(os.Stderr, "No templates found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Showing %s of %s templates\n\n", result.ResultSetSize, result.TotalSetSize)

	for _, tmpl := range result.EnvelopeTemplates {
		fmt.Printf("ID:   %s\n", tmpl.TemplateID)
		fmt.Printf("Name: %s\n", tmpl.Name)

		if tmpl.Description != "" {
			fmt.Printf("Desc: %s\n", tmpl.Description)
		}

		if tmpl.LastModified != "" {
			fmt.Printf("Modified: %s\n", tmpl.LastModified)
		}

		fmt.Println()
	}

	return nil
}

type TemplatesGetCmd struct {
	ID string `arg:"" required:"" help:"Template ID"`
}

func (cmd *TemplatesGetCmd) Run(ctx context.Context) error {
	client, err := getDocuSignClient(ctx)
	if err != nil {
		return err
	}

	result, err := client.GetTemplate(ctx, cmd.ID)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}
	if outfmt.IsPlain(ctx) {
		return outfmt.WritePlain(os.Stdout,
			[]string{"ID", "NAME", "DESCRIPTION", "CREATED", "MODIFIED"},
			[][]string{{result.TemplateID, result.Name, result.Description, result.Created, result.LastModified}},
		)
	}

	fmt.Printf("ID:   %s\n", result.TemplateID)
	fmt.Printf("Name: %s\n", result.Name)

	if result.Description != "" {
		fmt.Printf("Desc: %s\n", result.Description)
	}

	if result.Created != "" {
		fmt.Printf("Created:  %s\n", result.Created)
	}

	if result.LastModified != "" {
		fmt.Printf("Modified: %s\n", result.LastModified)
	}

	return nil
}
