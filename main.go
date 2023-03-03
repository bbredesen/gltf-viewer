package main

import (
	"fmt"
	"os"

	"github.com/bbredesen/gltf"
	"github.com/bbredesen/gltf-viewer/shared"
)

func init() {

}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s filename.gltf\n", os.Args[0])
		os.Exit(1)
	}

	gltfInput, err := gltf.FromFilename(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file %s: %s\n", os.Args[1], err.Error())
		os.Exit(1)
	}

	gltfDoc, err := gltfInput.Resolve(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error processing file %s: %s\n", os.Args[1], err.Error())
		os.Exit(1)
	}

	app := NewApp()
	app.Initialize() // Move pipeline creation to after loadGlTF, or as part of it?
	// Opt b is to have a standard buffer format for position, color, etc. and translate from the format in the file?
	// Translation is not always required. See spec section 3.7.2, attribute types have semantics for acessor and component types, eg. position is
	// always a VEC3 of FLOAT. Some have multiple options. e.g. COLOR_n can be vec3 or vec4, color components can be float, byte
	// or short. In that case though, byte or short are normalized, so you could divide by the max value to get a
	// floating point result in the range (-1..1)

	if err := app.loadGlTF(gltfDoc); err != nil {
		fmt.Fprintf(os.Stderr, "error loading glTF to graphics engine: %s\n", err.Error())
	}

	app.winapp.DefaultMainLoop(shared.DefaultIgnoreInput, shared.DefaultIgnoreTick, app.drawFrame)

	app.Teardown()
}
