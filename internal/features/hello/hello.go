// Package hello greets someone: barcost hello <name>.
package hello

// Name is the feature's command name.
const Name = "hello"

// Run is the greeting for the first argument, or for the world.
func Run(args []string) (string, error) {
	who := "World"
	if len(args) > 0 {
		who = args[0]
	}
	return "Hello, " + who + "!", nil
}
