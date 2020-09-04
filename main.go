package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PDXbaap/go-std-ext/statik"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strings"
	"sync"
	"syscall"
	"text/template"
)

var (
	supports = []string{
		"go1.14.4",
		"go1.14.5",
		"go1.14.6",
		"go1.14.7",
		"go1.14.8",
		"go1.15",
		"go1.15.1",
	}
	vsn2tag = func(vsn string) string { return strings.ReplaceAll(vsn, ".", "_") }
	app     = cli.NewApp()
)

type DictItem struct {
	Output string `json:"output"`
	Source string `json:"source"`
	Md5    string `json:"md5"`
	Action string `json:"action"`
}

type Dict []*DictItem

func (d *Dict) List(callback func(item *DictItem) error) {
	for _, it := range *d {
		if err := callback(it); err != nil {
			//fmt.Println(err)
			return
		}
	}
}

func init() {
	app.Name = os.Args[0]
	app.Usage = "PDX Stdlib 扩展"
	app.Version = "0.0.3"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "goroot,g",
			Usage: "go env 中 GOROOT 对应的目录, pdx 的扩展会被解压到这个目录中",
		},
		&cli.BoolFlag{
			Name:  "revert,r",
			Usage: "还原标准库",
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:   "rebuild",
			Action: rebuild,
		},
	}
	app.Action = action
}

// statik -src public/go1.14.4 -dest statik -p go1_14_4
func rebuild(_ *cli.Context) error {
	dir, _ := os.Getwd()
	fis, err := ioutil.ReadDir(path.Join(dir, "public"))
	if err != nil {
		panic(err)
	}
	var tags = make([]string, 0)
	for _, f := range fis {
		_vsn := f.Name()
		_tag := strings.ReplaceAll(_vsn, ".", "_")
		rtn, err := syncexec("statik", "-src", fmt.Sprintf("public/%s", _vsn), "-dest", "statik", "-p", _tag)
		if err != nil {
			panic(err)
		}
		codePath := path.Join(dir, "statik", _tag, "statik.go")
		fmt.Println(string(rtn), codePath)
		code, err := ioutil.ReadFile(codePath)
		if err != nil {
			panic(err)
		}
		//fs.RegisterWithNamespace("go1_15",data)
		code = bytes.ReplaceAll(code, []byte("fs.Register("), []byte(fmt.Sprintf("fs.RegisterWithNamespace(\"%s\", ", _tag)))
		err = ioutil.WriteFile(codePath, code, 00755)
		if err != nil {
			panic(err)
		}
		tags = append(tags, _tag)
	}
	utilCode := utilTempInst(tags)
	err = ioutil.WriteFile(path.Join(dir, "statik", "utils.go"), []byte(utilCode), 00755)
	if err != nil {
		panic(err)
	}

	return nil
}

func basepath() string {
	rtn, err := syncexec("go", "env")
	if err != nil {
		panic(err)
	}
	for _, kv := range strings.Split(string(rtn), "\n") {
		if strings.Contains(kv, "GOROOT") {
			return path.Join(strings.ReplaceAll(strings.Split(kv, "=")[1], "\"", ""), "src")
		}
	}
	return ""
}

func action(ctx *cli.Context) error {
	base := basepath()
	if ctx.IsSet("goroot") {
		base = path.Join(ctx.String("goroot"), "src")
	}
	fmt.Println("GOROOT : ", base)
	// 如果文件不存在，则返回错误
	fileInfo, err := os.Stat(base)
	if err != nil {
		log.Fatal(err)
	}
	ss := fileInfo.Sys().(*syscall.Stat_t)
	uu, err := user.Current()
	if err != nil {
		panic(err)
	}
	if fmt.Sprintf("%d", ss.Uid) != uu.Uid && fmt.Sprintf("%d", ss.Gid) != uu.Gid {
		err = errors.New(base + " dir no permision")
		log.Fatal(err)
		return err
	}
	return setup(base, ctx.Bool("revert"))
}

func setup(base string, revert bool) error {
	vsn, err := syncexec("go", "version")
	if err != nil {
		panic(err)
	}
	fmt.Println("VERSION : ", string(vsn))

	for _, _v := range supports {
		if strings.Contains(string(vsn), _v) {
			fs, err := statik.GetFileSystem(vsn2tag(_v))
			if err != nil {
				return err
			}
			f1, err := fs.Open("/dict.json")
			if err != nil {
				panic(err)
			}
			defer func() { _ = f1.Close() }()
			_, err = f1.Stat()
			if err != nil {
				panic(err)
			}

			j, err := ioutil.ReadAll(f1)
			if err != nil {
				panic(err)
			}
			var dict Dict
			_ = json.Unmarshal(j, &dict)
			dict.List(func(item *DictItem) error {
				//fmt.Println(path.Join(base, item.Output), item.Md5)
				if item.Action == "modify" {
					if revert {
						of, err := os.Open(path.Join(base, item.Output+".old"))
						if err != nil {
							panic(err)
						}
						data, err := ioutil.ReadAll(of)
						if err != nil {
							panic(err)
						}
						//fmt.Println(path.Join(base, item.Output))
						err = ioutil.WriteFile(path.Join(base, item.Output), data, 0644)
						if err != nil {
							panic(err)
						}
						err = os.Remove(path.Join(base, item.Output+".old"))
						if err != nil {
							panic(err)
						}
					} else {
						fout, err := ioutil.ReadFile(path.Join(base, item.Output))
						if err != nil {
							panic(err)
						}
						m5 := md5.New()
						m5.Write(fout)
						h := m5.Sum(nil)
						if hex.EncodeToString(h) != item.Md5 {
							// backoff
							if err = os.Rename(path.Join(base, item.Output), path.Join(base, item.Output+".old")); err != nil {
								panic(err)
							}
						}
						fsrc, _ := fs.Open(path.Join("/", item.Source))
						defer fsrc.Close()
						srcdata, _ := ioutil.ReadAll(fsrc)
						if err = ioutil.WriteFile(path.Join(base, item.Output), srcdata, 0644); err != nil {
							panic(err)
						}
					}
				} else if item.Action == "add" {
					if revert {
						_ = os.Remove(path.Join(base, item.Output))
					} else {
						fsrc, _ := fs.Open(path.Join("/", item.Source))
						defer fsrc.Close()
						srcdata, _ := ioutil.ReadAll(fsrc)
						if err = ioutil.WriteFile(path.Join(base, item.Output), srcdata, 0644); err != nil {
							panic(err)
						}
					}
				}
				return nil
			})
			return nil
		}
	}
	/*
		fs, err := fs.New()
		if err != nil {
			panic(err)
		}
		f, err := fs.Open("/")
		defer f.Close()
		if err != nil {
			panic(err)
		}
		ff, err := f.Readdir(0)
		if err != nil {
			panic(err)
		}

		for _, f := range ff {
			if strings.Contains(string(vsn), f.Name()) {
				f1, err := fs.Open(path.Join("/", f.Name(), "dict.json"))
				if err != nil {
					panic(err)
				}
				defer func() { _ = f1.Close() }()
				_, err = f1.Stat()
				if err != nil {
					panic(err)
				}
				//fmt.Println(f1Stat.IsDir(), " ", f1Stat.Name())
				j, err := ioutil.ReadAll(f1)
				if err != nil {
					panic(err)
				}
				var dict Dict
				_ = json.Unmarshal(j, &dict)
				dict.List(func(item *DictItem) error {
					//fmt.Println(path.Join(base, item.Output), item.Md5)
					if item.Action == "modify" {
						if revert {
							of, err := os.Open(path.Join(base, item.Output+".old"))
							if err != nil {
								panic(err)
							}
							data, err := ioutil.ReadAll(of)
							if err != nil {
								panic(err)
							}
							//fmt.Println(path.Join(base, item.Output))
							err = ioutil.WriteFile(path.Join(base, item.Output), data, 0644)
							if err != nil {
								panic(err)
							}
							err = os.Remove(path.Join(base, item.Output+".old"))
							if err != nil {
								panic(err)
							}
						} else {
							fout, err := ioutil.ReadFile(path.Join(base, item.Output))
							if err != nil {
								panic(err)
							}
							m5 := md5.New()
							m5.Write(fout)
							h := m5.Sum(nil)
							if hex.EncodeToString(h) != item.Md5 {
								// backoff
								if err = os.Rename(path.Join(base, item.Output), path.Join(base, item.Output+".old")); err != nil {
									panic(err)
								}
							}
							fsrc, _ := fs.Open(path.Join("/", f.Name(), item.Source))
							defer fsrc.Close()
							srcdata, _ := ioutil.ReadAll(fsrc)
							if err = ioutil.WriteFile(path.Join(base, item.Output), srcdata, 0644); err != nil {
								panic(err)
							}
						}
					} else if item.Action == "add" {
						if revert {
							_ = os.Remove(path.Join(base, item.Output))
						} else {
							fsrc, _ := fs.Open(path.Join("/", f.Name(), item.Source))
							defer fsrc.Close()
							srcdata, _ := ioutil.ReadAll(fsrc)
							if err = ioutil.WriteFile(path.Join(base, item.Output), srcdata, 0644); err != nil {
								panic(err)
							}
						}
					}

					return nil
				})
				return nil
			}
		}
	*/
	fmt.Println(">###### support list ######>")
	for _, v := range supports {
		fmt.Println(v)
	}
	fmt.Println("<###### support list ######<")
	return fmt.Errorf("%v not support yet", vsn)
}

func syncexec(bin string, args ...string) (rtn []byte, err error) {
	cmd := exec.Command(bin, args...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		rtn, err = ioutil.ReadAll(out)
		//fmt.Println(string(rtn))
	}()
	if err = cmd.Run(); err != nil {
		return
	}
	wg.Wait()
	return
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Println("Success.")
}

func utilTempInst(al []string) string {
	var buf = new(bytes.Buffer)
	_ = utilTemp.Execute(buf, map[string]interface{}{"DataList": al})
	return buf.String()
}

var utilTemp, _ = template.New("util").Parse(`package statik
import (
{{range $idx, $itm := .DataList}}
	_ "github.com/PDXbaap/go-std-ext/statik/{{$itm}}"
{{end}}
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
`)
