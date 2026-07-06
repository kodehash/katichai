package cmd

import (
	"github.com/katichai/katich/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start katich as an MCP tool server (stdio)",
	Long: `Run katich as a Model Context Protocol (MCP) server over stdio.

This exposes katich commands (review, context, doctor) as callable tools for
AI agents such as Claude Code, OpenAI Codex, and Cursor.

The server reads JSON-RPC requests from stdin and writes responses to stdout.
It is not intended to be run manually — configure it in your MCP client's
server list instead.

Example .mcp.json entry:
  {
    "mcpServers": {
      "katich": {
        "command": "katich",
        "args": ["mcp"]
      }
    }
  }`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return mcp.Serve()
	},
}
