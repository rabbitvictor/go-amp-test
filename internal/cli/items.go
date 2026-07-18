package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newItemsCmd(flags *rootFlags, out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "items",
		Short: "Manage items",
		Long:  "Subcommands for listing, getting, and creating items via the web API.",
	}
	cmd.AddCommand(newItemsListCmd(flags, out, errOut))
	cmd.AddCommand(newItemsGetCmd(flags, out, errOut))
	cmd.AddCommand(newItemsCreateCmd(flags, out, errOut))
	return cmd
}

func newItemsListCmd(flags *rootFlags, out, errOut io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all items",
		Long:  "Calls GET /items and prints the list as JSON.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := flags.newAPIClient().ListItems(cmd.Context())
			if err != nil {
				return err
			}
			return writeOut(out, flags.outFormat(), items)
		},
	}
}

func newItemsGetCmd(flags *rootFlags, out, errOut io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a single item by id",
		Long:  "Calls GET /items/:id and prints the item as JSON.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return newUsageError(fmt.Errorf("invalid id %q: %w", args[0], err))
			}
			it, err := flags.newAPIClient().GetItem(cmd.Context(), id)
			if err != nil {
				return err
			}
			return writeOut(out, flags.outFormat(), it)
		},
	}
}

func newItemsCreateCmd(flags *rootFlags, out, errOut io.Writer) *cobra.Command {
	var (
		name string
		data string // raw JSON body, or "-" to read from stdin
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new item",
		Long: `Create a new item via POST /items.

By default the item name is taken from --name. To create an item from raw JSON
(for example with additional future fields), pass --data with the JSON body, or
--data - to read the body from stdin:

	go-amp-test items create --name "my item"
	echo '{"name":"from stdin"}' | go-amp-test items create --data -`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := resolveName(name, data)
			if err != nil {
				return newUsageError(err)
			}
			if resolved == "" {
				return newUsageError(errors.New("name is required (use --name or --data)"))
			}
			it, err := flags.newAPIClient().CreateItem(cmd.Context(), resolved)
			if err != nil {
				return err
			}
			fmt.Fprintf(errOut, "created item %d\n", it.ID)
			return writeOut(out, flags.outFormat(), it)
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "item name")
	cmd.Flags().StringVar(&data, "data", "", "raw JSON body, or '-' to read from stdin")
	cmd.MarkFlagsMutuallyExclusive("name", "data")
	return cmd
}

// resolveName picks the item name from --name or --data. When data is "-",
// the JSON body is read from stdin and the "name" field is extracted.
func resolveName(name, data string) (string, error) {
	if data == "" {
		return name, nil
	}
	raw := data
	if data == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		raw = strings.TrimSpace(string(b))
	}
	var in struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		return "", fmt.Errorf("parse --data JSON: %w", err)
	}
	return in.Name, nil
}
