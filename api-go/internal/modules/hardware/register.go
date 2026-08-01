package hardware

import "github.com/XoDeR/empops/api-go/pkg/module"

func init() {
	module.DefaultRegistry.Register(New())
}
