package mcpserver

import (
	"context"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/mark3labs/mcp-go/mcp"
)

func addDocumentTool() mcp.Tool {
	return mcp.NewTool(types.MCPEndpointToolAddDocument,
		mcp.WithDescription("Add a document. Write execution is disabled in the initial Xelora port."),
		mcp.WithString("knowledge_base_id", mcp.Required()),
		mcp.WithString("title", mcp.Required()),
		mcp.WithString("content"),
		mcp.WithDestructiveHintAnnotation(true),
	)
}

func updateDocumentTool() mcp.Tool {
	return mcp.NewTool(types.MCPEndpointToolUpdateDocument,
		mcp.WithDescription("Update a document. Write execution is disabled in the initial Xelora port."),
		mcp.WithString("knowledge_id", mcp.Required()),
		mcp.WithString("content", mcp.Required()),
		mcp.WithDestructiveHintAnnotation(true),
	)
}

func deleteDocumentTool() mcp.Tool {
	return mcp.NewTool(types.MCPEndpointToolDeleteDocument,
		mcp.WithDescription("Delete a document. Write execution is disabled in the initial Xelora port."),
		mcp.WithString("knowledge_id", mcp.Required()),
		mcp.WithDestructiveHintAnnotation(true),
	)
}

func (s *Server) handleAddDocument(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError("write operations are not enabled for this MCP endpoint"), nil
}
func (s *Server) handleUpdateDocument(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError("write operations are not enabled for this MCP endpoint"), nil
}
func (s *Server) handleDeleteDocument(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError("write operations are not enabled for this MCP endpoint"), nil
}
