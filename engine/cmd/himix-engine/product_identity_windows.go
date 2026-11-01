//go:build windows

package main

// Product identity for the installed layout. The installer (NSIS) and the
// engine must agree on these names byte for byte: the engine refuses to run
// as the Core Service unless its own executable sits at exactly the path the
// installer stages, and the desktop host handshake compares executable names
// as well. Keep every product-name literal in the engine here so a rename
// cannot silently desynchronize the two sides.
const (
	// productDirectoryName is the per-machine ProgramData directory that
	// holds the protected Core Service payload.
	productDirectoryName = "Himix"

	// engineExecutableName is the file name of the Core Service executable
	// inside the protected bin directory.
	engineExecutableName = "himix-engine.exe"

	// desktopExecutableName is the file name of the desktop application that
	// hosts the Core Service.
	desktopExecutableName = "himix.exe"

	// coreServiceDisplayName is the user-visible name of the Windows service.
	coreServiceDisplayName = "Himix Core Service"
)
