package utils

import (
	"github.com/go-rod/rod"
	"github.com/agenthands/rod-cli/types/js"
)

func QueryEleByAria(frame *rod.Page, selector string) (*rod.Element, error) {
	return frame.ElementByJS(
		rod.Eval(js.QueryEleByAria, selector),
	)
}
