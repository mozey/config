// +build !windows

// TODO Consider renaming this to x_linux.go,
// and windows.go to x_windows.go, see this
// https://stackoverflow.com/a/25162021/639133

// OS Detection at compile time
// https://stackoverflow.com/a/19847868/639133
package cmdconfig

const LineBreak = "\n"
const ExportFormat = "export %v=%v"
const UnsetFormat = "unset %v"
