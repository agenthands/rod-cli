package tools

import (
	"context"
	"github.com/charmbracelet/log"
	"github.com/agenthands/rod-cli/actions"
	"github.com/agenthands/rod-cli/types"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	NavigationToolKey   = "rod_navigate"
	GoBackToolKey       = "rod_go_back"
	GoForwardToolKey    = "rod_go_forward"
	ReloadToolKey       = "rod_reload"
	PressKeyToolKey     = "rod_press"
	PdfToolKey          = "rod_pdf"
	ScreenshotToolKey   = "rod_screenshot"
	EvaluateToolKey     = "rod_evaluate"
	CloseBrowserToolKey = "rod_close_browser"
)

var (
	Navigation = mcp.NewTool("rod_navigate",
		mcp.WithDescription("Navigate to a URL"),
		mcp.WithString("url", mcp.Description("URL to navigate to"), mcp.Required()),
	)
	GoBack = mcp.NewTool(GoBackToolKey,
		mcp.WithDescription("Go back in the browser history, go back to the previous page"),
	)
	GoForward = mcp.NewTool(GoForwardToolKey,
		mcp.WithDescription("Go forward in the browser history, go to the next page"),
	)
	ReLoad = mcp.NewTool(ReloadToolKey,
		mcp.WithDescription("Reload the current page"),
	)
	PressKey = mcp.NewTool(PressKeyToolKey,
		mcp.WithDescription("Press a key on the keyboard"),
		mcp.WithString("key", mcp.Description("Name of the key to press or a character to generate, such as `ArrowLeft` or `a`"), mcp.Required()),
	)
	Pdf = mcp.NewTool(PdfToolKey,
		mcp.WithDescription("Generate a PDF from the current page"),
		mcp.WithString("file_path", mcp.Description("Path to save the PDF file"), mcp.Required()),
		mcp.WithString("file_name", mcp.Description("Name of the PDF file"), mcp.Required()),
	)
	CloseBrowser = mcp.NewTool(CloseBrowserToolKey,
		mcp.WithDescription("Close the browser"),
	)
	Screenshot = mcp.NewTool(ScreenshotToolKey,
		mcp.WithDescription("Take a screenshot of the current page or a specific element"),
		mcp.WithString("name", mcp.Description("Name of the screenshot"), mcp.Required()),
		mcp.WithString("selector", mcp.Description("CSS selector of the element to take a screenshot of")),
		mcp.WithNumber("width", mcp.Description("Width in pixels (default: 800)")),
		mcp.WithNumber("height", mcp.Description("Height in pixels (default: 600)")),
	)
	Evaluate = mcp.NewTool(EvaluateToolKey,
		mcp.WithDescription("Execute JavaScript in the browser console"),
		mcp.WithString("script", mcp.Description("A function name or an unnamed function definition"), mcp.Required()),
	)
)

var (
	NavigationHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			url := request.Params.Arguments["url"].(string)
			msg, err := actions.Navigate(rodCtx, url)
			if err != nil {
				log.Errorf("Navigate error: %v", err)
				return nil, err
			}
			return mcp.NewToolResultText(msg), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	GoBackHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			msg, err := actions.GoBack(rodCtx)
			if err != nil {
				log.Errorf("GoBack error: %v", err)
				return nil, err
			}
			return mcp.NewToolResultText(msg), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	GoForwardHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			msg, err := actions.GoForward(rodCtx)
			if err != nil {
				log.Errorf("GoForward error: %v", err)
				return nil, err
			}
			return mcp.NewToolResultText(msg), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	ReLoadHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			msg, err := actions.Reload(rodCtx)
			if err != nil {
				log.Errorf("Reload error: %v", err)
				return nil, err
			}
			return mcp.NewToolResultText(msg), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	PressKeyHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			keyStr := request.Params.Arguments["key"].(string)
			key := []rune(keyStr)[0]
			msg, err := actions.PressKey(rodCtx, key)
			if err != nil {
				log.Errorf("PressKey error: %v", err)
				return nil, err
			}
			return mcp.NewToolResultText(msg), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: rodCtx.CurrentMode() == types.Text})
	}

	CloseBrowserHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			msg, err := actions.CloseBrowser(rodCtx)
			if err != nil {
				log.Errorf("CloseBrowser error: %v", err)
				return nil, err
			}
			return mcp.NewToolResultText(msg), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: false})
	}

	EvaluateHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			script := request.Params.Arguments["script"].(string)
			msg, err := actions.Evaluate(rodCtx, script, "")
			if err != nil {
				log.Errorf("Evaluate error: %v", err)
				return nil, err
			}
			return mcp.NewToolResultText(msg), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: false})
	}

	ScreenshotHandler = func(rodCtx *types.Context) server.ToolHandlerFunc {
		handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := request.Params.Arguments["name"].(string)
			var selector string
			if sel, ok := request.Params.Arguments["selector"].(string); ok {
				selector = sel
			}
			var width, height float64
			if w, ok := request.Params.Arguments["width"].(float64); ok {
				width = w
			}
			if h, ok := request.Params.Arguments["height"].(float64); ok {
				height = h
			}
			msg, err := actions.Screenshot(rodCtx, name, selector, width, height)
			if err != nil {
				log.Errorf("Screenshot error: %v", err)
				return nil, err
			}
			return mcp.NewToolResultText(msg), nil
		}
		return rodCtx.Execute(handler, types.ToolHandlerCallOpts{WitSnapshot: false})
	}
)

var (
	CommonTools = []mcp.Tool{
		Navigation,
		GoBack,
		GoForward,
		ReLoad,
		PressKey,
		Screenshot,
		Evaluate,
		CloseBrowser,
	}
	CommonToolHandlers = map[string]ToolHandler{
		NavigationToolKey:   NavigationHandler,
		GoBackToolKey:       GoBackHandler,
		GoForwardToolKey:    GoForwardHandler,
		ReloadToolKey:       ReLoadHandler,
		PressKeyToolKey:     PressKeyHandler,
		ScreenshotToolKey:   ScreenshotHandler,
		EvaluateToolKey:     EvaluateHandler,
		CloseBrowserToolKey: CloseBrowserHandler,
	}
)
