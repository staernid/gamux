//go:build windows && 386

package steamless

import _ "embed"

//go:embed bin/steamless-windows-x86.exe
var embeddedBinary []byte

const embeddedBinaryName = "steamless.exe"
