package tools

import (
	"context"
	"github.com/charmbracelet/log"
	"github.com/agenthands/rod-cli/actions"
	"github.com/agenthands/rod-cli/types"
	"github.com/agenthands/rod-cli/utils"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	SnapshotToolKey = "rod_snapshot"
	ClickToolKey    = "rod_click"
	FillToolKey     = "rod_fill"
	SelectorToolKey = "rod_selector"
)

var (
	Snapshot = mcp.NewTool("rod_snapshot",
		mcp.WithDescription("Capture accessibility snapshot of the current page, this is better than screenshot"),
	)

	Click = mcp.NewTool(ClickToolKey,
		mcp.WithDescription("Perform click on a web page"),
		mcp.WithString("element", mcp.Description("Human-readable element description used to obtain permission to interact with the element"), mcp.Required()),
		mcp.WithString("ref", mcp.Description("Exact target element reference from the page snapshot"), mcp.Required()),
	)

	Fill = mcp.NewTool(FillToolKey,
		mcp.WithDescription("Type text into editable element"),
		mcp.WithString("element", mcp.Description("Human-readable element description used to obtain permission to interact with the element"), mcp.Required()),
		mcp.WithString("value", mcp.Description("Text to type into the element"), mcp.Required()),
		mcp.WithString("ref", mcp.Description("Exact target element reference from the page snapshot"), mcp.Required()),
		mcp.WithBoolean("submit", mcp.Description("Whether to type one character at a time. Useful for triggering key handlers in the page. By default entire text is filled in at once."), mcp.Required()),
	)
	Selector = mcp.NewTool(SelectorToolKey,
		mcp.WithDescription("Select an option in a dropdown"),
		mcp.WithString("element", mcp.Description("Human-readable element description used to obtain permission to interact with the element"), mcp.Required()),
		mcp.WithString("ref", mcp.Description("Exact target element reference from the page snapshot"), mcp.Required()),
		mcp.WithArray("values", mcp.Description("Array of values to select in the dropdown. This can be a single value or multiple values."), mcp.Items(map[string]interface{}{"type": "string", "required": true}), mcp.Required()),
	)
)

var (
	SnapshotHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			snapshot, err := actions.Snapshot(rodCtx)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(snapshot), nil

		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: false})
	}

	ClickHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ref := request.Params.Arguments["ref"].(string)
			msg, err := actions.Click(rodCtx, ref)
			if err != nil {
				log.Errorf("Click error: %v", err)
				return nil, err
			}
			return mcp.NewToolResultText(msg), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: true})
	}

	FillHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ref := request.Params.Arguments["ref"].(string)
			value := request.Params.Arguments["value"].(string)
			submit, _ := request.Params.Arguments["submit"].(bool)
			msg, err := actions.Fill(rodCtx, ref, value, submit)
			if err != nil {
				log.Errorf("Fill error: %v", err)
				return nil, err
			}
			return mcp.NewToolResultText(msg), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: true})
	}

	SelectorHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ref := request.Params.Arguments["ref"].(string)
			values, err := utils.OptionalStringArrayParam(request, "values")
			if err != nil {
				return nil, err
			}
			msg, err := actions.Select(rodCtx, ref, values)
			if err != nil {
				log.Errorf("Select error: %v", err)
				return nil, err
			}
			return mcp.NewToolResultText(msg), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: true})
	}
)

var (
	SnapshotToolHandlers = map[string]ToolHandler{
		SnapshotToolKey: SnapshotHandler,
		ClickToolKey:    ClickHandler,
		FillToolKey:     FillHandler,
		SelectorToolKey: SelectorHandler,
	}
	Snapshots = []mcp.Tool{
		Snapshot,
		Click,
		Fill,
		Selector,
	}
)
