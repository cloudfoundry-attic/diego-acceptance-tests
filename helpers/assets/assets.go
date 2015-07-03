package assets

type Assets struct {
	Dora       string
	HelloWorld string
	Standalone string
	Fuse       string
	Node       string
	Java       string
	Python     string
	Php        string
	Staticfile string
	Golang     string
}

func NewAssets() Assets {
	return Assets{
		Dora:       "../assets/dora",
		HelloWorld: "../assets/hello-world",
		Standalone: "../assets/standalone",
		Fuse:       "../assets/fuse-mount",
		Node:       "../assets/node",
		Golang:     "../assets/golang",
		Java:       "../assets/java",
		Python:     "../assets/python",
		Php:        "../assets/php",
		Staticfile: "../assets/staticfile",
	}
}
