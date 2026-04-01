package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/infrastructure/config"
)

var requirementCompleteCmd = &cobra.Command{
	Use:   "complete",
	Short: "完成需求",
	Example: `  taskmanager requirement complete --id <id>`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")

		if id == "" {
			fmt.Print(`{"error":"--id is required"}`)
			return
		}

		token, err := login(defaultAdminUsername, defaultAdminPassword)
		if err != nil {
			fmt.Printf(`{"error":"login failed: %v"}`, err)
			return
		}

		reqBody := map[string]string{
			"requirement_id": id,
		}
		reqJSON, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", config.GetAPIBaseURL()+"/requirements/pr", bytes.NewBuffer(reqJSON))
		if err != nil {
			fmt.Printf(`{"error":"create request failed: %v"}`, err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf(`{"error":"complete request failed: %v"}`, err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			fmt.Printf(`{"error":"complete failed: %s","detail":%s}`, resp.Status, string(body))
			return
		}

		fmt.Print(string(body))
	},
}

func init() {
	requirementCompleteCmd.Flags().StringP("id", "i", "", "需求 ID (必填)")
}
