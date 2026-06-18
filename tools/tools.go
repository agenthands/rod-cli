package tools

import (
	"github.com/agenthands/rod-cli/types"
	"github.com/agenthands/rod-cli/utils"
	"github.com/mark3labs/mcp-go/server"
)

type ToolHandler = func(rodCtx *types.Context) server.ToolHandlerFunc

var (
	TextTools        = append(CommonTools, Snapshots...)
	TextToolHandlers = utils.MergeMaps(CommonToolHandlers, SnapshotToolHandlers)
)
