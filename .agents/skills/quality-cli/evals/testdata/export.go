package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type item struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Updated time.Time `json:"updated"`
}

// newExportCmd implements `shipit export <project>`: it downloads all items of a
// project, prints them, saves the last export in ~/.shipit.json and optionally
// deletes the exported items from the server.
func newExportCmd() *cobra.Command {
	var force bool
	var format string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export items",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("error: project name required")
				os.Exit(0)
			}

			resp, err := http.Get("https://api.shipit.example/v1/projects/" + args[0] + "/items")
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			var items []item
			if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
				log.Fatal(err)
			}

			if len(items) == 0 {
				fmt.Println("No items found!")
				os.Exit(1)
			}

			for _, it := range items {
				fmt.Printf("\033[32m%s\033[0m  %s  %s ago\n", it.ID, it.Name, time.Since(it.Updated).Round(time.Hour))
			}

			data, _ := json.Marshal(map[string]any{
				"project": args[0],
				"items":   items,
				"token":   os.Getenv("SHIPIT_TOKEN"),
			})
			ioutil.WriteFile(os.Getenv("HOME")+"/.shipit.json", data, 0644)

			if !force {
				fmt.Print("Delete exported items from the server? [y/N] ")
				answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				if answer != "y\n" {
					fmt.Println("Done!")
					return
				}
			}
			for _, it := range items {
				req, _ := http.NewRequest(http.MethodDelete, "https://api.shipit.example/v1/items/"+it.ID, nil)
				http.DefaultClient.Do(req)
			}
			fmt.Println("Done!")
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "")
	cmd.Flags().StringVar(&format, "format", "table", "output format")
	return cmd
}
