module github.com/bbredesen/gltf-viewer

go 1.20

replace (
	github.com/bbredesen/gltf v0.0.1 => C:\Users\benbr\go\src\github.com\bbredesen\gltf
	github.com/bbredesen/vkm v0.2.0 => C:\Users\benbr\go\src\github.com\bbredesen\vkm
	github.com/bbredesen/win32-toolkit v0.0.1 => C:\Users\benbr\go\src\github.com\bbredesen\win32-toolkit
)

require (
	github.com/bbredesen/gltf v0.0.1
	github.com/bbredesen/go-vk v0.0.0-20230217143317-ce5fce0dc2f2
	github.com/bbredesen/vkm v0.2.0
	github.com/bbredesen/win32-toolkit v0.0.1
	golang.org/x/sys v0.4.0
)

require github.com/chewxy/math32 v1.10.1
