package statik
import (

	_ "github.com/PDXbaap/go-std-ext/statik/go1_14_4"

	_ "github.com/PDXbaap/go-std-ext/statik/go1_14_5"

	_ "github.com/PDXbaap/go-std-ext/statik/go1_14_6"

	_ "github.com/PDXbaap/go-std-ext/statik/go1_14_7"

	_ "github.com/PDXbaap/go-std-ext/statik/go1_14_8"

	_ "github.com/PDXbaap/go-std-ext/statik/go1_14_9"

	_ "github.com/PDXbaap/go-std-ext/statik/go1_15"

	_ "github.com/PDXbaap/go-std-ext/statik/go1_15_1"

	_ "github.com/PDXbaap/go-std-ext/statik/go1_15_2"

	"github.com/rakyll/statik/fs"
	"net/http"
)
func GetFileSystem(tag string) (http.FileSystem, error) {
	f, err := fs.NewWithNamespace(tag)
	if err != nil {
		return nil, err
	}
	return f, nil
}
