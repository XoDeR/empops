package example

import "github.com/XoDeR/empops/api-go/pkg/module"

// init registers the example module into the default registry so that
// blank-importing this package (as cmd/api does) is enough to make it
// available for enabling via config/modules.yaml.
func init() {
	module.DefaultRegistry.Register(New())
}
