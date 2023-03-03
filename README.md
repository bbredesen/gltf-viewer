# glTF-viewer

A simple model viewer for glTF files in Go on Windows. Work in progress, but currently rendering geometry to the screen. Support
for model colors, textures, cameras, animation, etc. remains to be done. 

## Development Status
The real purpose of this project at the moment is to test and work out bugs in
[go-vk](https://github.com/bbredesen/go-vk), which is itself in a *very* alpha state. The project currently uses
`replace` clauses in go.mod to point at local copies of some libraries (vector math, gltf loader, and
win32 support). 

This code is not clean and has lots of "work in progress" comments, but it is working as a proof of concept.

# License
MIT license. See the LICENSE file.

SPDX-License-Identifier: MIT

