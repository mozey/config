// +build windows

package cmdconfig

const LineBreak = "\r\n"

// Note the difference between set and setx
// https://superuser.com/a/916652/537059
const ExportFormat = "set %v=%v"
const UnsetFormat = "set %v=\"\""