package cmdconfig

// This file defines cross-platform config,
// the corresponding "x_${GOOS}.go" file must set values appropriate for GOOS

// Files ending with "_${GOOS}.go" will only build for that OS,
// see this https://stackoverflow.com/a/25162021/639133

// OS Detection at compile time
// https://stackoverflow.com/a/19847868/639133

// .............................................................................
// Windows

// Note the difference between set and setx
// https://superuser.com/a/916652/537059
const WindowsExportFormat = "set %v=%v"
const WindowsUnsetFormat = "set %v=\"\""
const WindowsLineBreak = "\r\n"

// .............................................................................
// Other

const OtherExportFormat = "export %v=%v"
const OtherUnsetFormat = "unset %v"
const OtherLineBreak = "\n"
