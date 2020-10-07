// +build !windows

// OS Detection at compile time
// https://stackoverflow.com/a/19847868/639133
package cmdconfig

const ExportFormat = "export %v=%v"
const UnsetFormat = "unset %v"