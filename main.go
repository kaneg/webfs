package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kaneg/flaskgo"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type WebFS struct {
	app *flaskgo.App
}

type FolderMeta struct {
	BaseName,
	Dir,
	Folder string
	Files []FileMeta
}

type FileMeta struct {
	Name    string
	IsDir   bool
	Size    int64
	ModTime string
}

func toFileMetas(infos []os.FileInfo) []FileMeta {
	var fileMetas = make([]FileMeta, len(infos))
	for i, info := range infos {
		fileMetas[i].Name = info.Name()
		fileMetas[i].IsDir = info.IsDir()
		fileMetas[i].Size = info.Size()
		fileMetas[i].ModTime = info.ModTime().Format(time.UnixDate)
	}
	return fileMetas
}

type ByName []os.FileInfo

func (a ByName) Len() int {
	return len(a)
}

func (a ByName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByName) Less(i, j int) bool {
	if a[i].IsDir() == a[j].IsDir() {
		return strings.ToLower(a[i].Name()) < strings.ToLower(a[j].Name())
	} else {
		return !a[i].IsDir()
	}
}

type BySize []os.FileInfo

func (a BySize) Len() int {
	return len(a)
}

func (a BySize) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a BySize) Less(i, j int) bool {
	if a[i].IsDir() && a[j].IsDir() {
		return strings.ToLower(a[i].Name()) < strings.ToLower(a[j].Name())
	} else if !a[i].IsDir() && !a[j].IsDir() {
		return a[i].Size() < a[j].Size()
	} else {
		return !a[i].IsDir()
	}
}

type ByTime []os.FileInfo

func (a ByTime) Len() int {
	return len(a)
}

func (a ByTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByTime) Less(i, j int) bool {
	if a[i].IsDir() == a[j].IsDir() {
		return a[i].ModTime().After(a[j].ModTime())
	} else {
		return !a[i].IsDir()
	}
}

func formatUserHome(filePath string) string {
	if filePath == "~" {
		usr, err := user.Current()
		if err != nil {
			return os.Getenv("HOME")
		}
		fmt.Println("User:", usr)
		filePath = usr.HomeDir
	}
	return filePath
}

func (fs *WebFS) normalizeInPath(filePath string) string {
	filePath = formatUserHome(filePath)

	if runtime.GOOS != "windows" {
		if !strings.HasPrefix(filePath, "/") {
			filePath = "/" + filePath
		}
	}
	filePath, _ = filepath.Abs(filePath)
	return filePath
}

func (fs *WebFS) normalizeOutPath(filePath string) string {
	filePath = formatUserHome(filePath)
	if runtime.GOOS != "windows" {
		if !strings.HasPrefix(filePath, "/") {
			filePath = "/" + filePath
		}
	}
	return filePath
}

func (fs *WebFS) SimpleList(filePath string) string {
	return fs.List("Name", true, filePath)
}

func (fs *WebFS) ListUp(filePath string) string {
	fmt.Println("Get up folder for:", filePath)
	filePath = fs.normalizeInPath(filePath)
	absPath, _ := filepath.Abs(filePath)
	upDir := filepath.Dir(absPath)
	fmt.Println("Up folder is :", upDir)
	return fs.SimpleList(upDir)
}

func (fs *WebFS) Index() string {
	c := make(flaskgo.Context)
	c["Dir"] = dir
	c["Prefix"] = fs.app.Prefix
	return fs.app.RenderTemplate("file_list.html", &c)
}

func (fs *WebFS) List(orderBy string, isAsc bool, filePath string) string {
	fmt.Println("List:" + filePath)
	filePath = fs.normalizeInPath(filePath)

	fmt.Println("Read filePath:" + filePath)
	dir, err := os.Open(filePath)
	defer dir.Close()
	if err == nil {
		info, err := dir.Stat()
		if err == nil {
			c := FolderMeta{}
			if info.IsDir() {
				files, _ := dir.Readdir(0)
				sortFile(&files, orderBy, isAsc)
				c.Files = toFileMetas(files)

				c.Dir = path.Dir(path.Dir(filePath))
				c.Folder = fs.normalizeOutPath(filePath)
				fmt.Println("Folder is1: " + filePath)
				fmt.Println("Folder is2: " + c.Folder)
				c.BaseName = path.Base(filePath)
				return returnJson(&c)
			} else {
				return returnSuccess("file:" + filePath)
			}
		}
	}

	return returnError("Failed to open file: " + filePath)
}

func returnError(msg string) string {
	return `{"success": false, "msg": "` + msg + `"}`
}

func returnSuccess(msg string) string {
	return `{"success": true, "msg": "` + msg + `"}`
}

func returnJson(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return returnError(err.Error())
	}
	return `{"success": true, "msg": ` + string(data) + `}`
}

func sortFile(files *[]os.FileInfo, orderBy string, isAsc bool) {
	var sortType sort.Interface
	switch orderBy {
	case "Name":
		sortType = ByName(*files)
	case "Size":
		sortType = BySize(*files)
	case "Time":
		sortType = ByTime(*files)
	}
	if !isAsc {
		sortType = sort.Reverse(sortType)
	}
	sort.Sort(sortType)
}

func errorJson(err error) string {
	if err != nil {
		return `{"status": "failed", "metadata":` + err.Error() + "}"
	} else {
		return `{"status": "success"}`
	}
}

func (fs *WebFS) MakeDirs(filePath string) string {
	filePath = fs.normalizeInPath(filePath)
	fmt.Println("Mkdir: " + filePath)

	err := os.MkdirAll(filePath, 0722)
	if err != nil {
		return returnError(err.Error())
	} else {
		return returnSuccess("")
	}
}

func (fs *WebFS) Remove(filePath string) string {
	filePath = fs.normalizeInPath(filePath)
	fmt.Println("Remove: " + filePath)

	err := os.Remove(filePath)

	if err != nil {
		log.Println(err)
		return returnError(err.Error())
	} else {
		return returnSuccess("")
	}
}

func (fs *WebFS) Download(filePath string) {
	filePath = fs.normalizeInPath(filePath)
	fmt.Println("get: " + filePath)
	fileName := path.Base(filePath)

	w := flaskgo.GetResponseWriter()
	in, err := os.OpenFile(filePath, os.O_RDONLY, 0)
	defer in.Close()
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte("Not Found"))
		return
	}
	info, _ := in.Stat()
	buffer := make([]byte, info.Size())
	in.Read(buffer)
	fmt.Println("read over")
	disposition := fmt.Sprintf("attachment; filename=\"%s\"", fileName)
	w.Header().Add("content-disposition", disposition)
	w.Header().Add("content-length", strconv.Itoa(int(info.Size())))
	w.Write(buffer)
	fmt.Println("write over")
}

func (fs *WebFS) View(filePath string) {
	filePath = fs.normalizeInPath(filePath)
	fmt.Println("get: " + filePath)
	w := flaskgo.GetResponseWriter()

	in, err := os.OpenFile(filePath, os.O_RDONLY, 0)
	defer in.Close()
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte("Not Found"))
		return
	}
	info, _ := in.Stat()
	buffer := make([]byte, info.Size())
	in.Read(buffer)
	fmt.Println("read over")
	w.Header().Add("content-type", "text/plain")
	w.Header().Add("content-length", strconv.Itoa(int(info.Size())))
	w.Write(buffer)
}

func (fs *WebFS) Edit(filePath string) {
	filePath = fs.normalizeInPath(filePath)
	fmt.Println("get: " + filePath)
	w := flaskgo.GetResponseWriter()

	in, err := os.OpenFile(filePath, os.O_RDONLY, 0)
	defer in.Close()
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte("Not Found"))
	}
	info, _ := in.Stat()
	buffer := make([]byte, info.Size())
	in.Read(buffer)

	fmt.Println("read over")
	c := make(flaskgo.Context)
	c["Content"] = string(buffer)
	c["FilePath"] = fs.normalizeOutPath(filePath)
	c["Folder"] = path.Dir(filePath)
	c["BaseName"] = path.Base(filePath)
	w.Write([]byte(fs.app.RenderTemplate("file_edit.html", &c)))
}
func (fs *WebFS) OnEdit(filePath string) string {
	filePath = fs.normalizeInPath(filePath)
	fmt.Println("get: " + filePath)
	in, err := os.OpenFile(filePath, os.O_RDONLY, 0)
	defer in.Close()
	if err != nil {
		log.Println(err)
		return returnError(err.Error())
	}
	info, _ := in.Stat()
	buffer := make([]byte, info.Size())
	in.Read(buffer)
	fmt.Println("read over")
	c := make(flaskgo.Context)
	c["Content"] = string(buffer)
	c["FilePath"] = fs.normalizeOutPath(filePath)
	c["Folder"] = path.Dir(filePath)
	c["BaseName"] = path.Base(filePath)
	return returnJson(&c)
}

func (fs *WebFS) Save(filePath string) string {
	fmt.Println("save: " + filePath)
	filePath = fs.normalizeInPath(filePath)
	r := flaskgo.GetRequest()

	content := r.FormValue("content")
	var buffer = []byte(content)

	out, err := os.Create(filePath)
	defer out.Close()
	if err != nil {
		log.Println(err)
		return returnError(err.Error())
	}
	_, err = out.Write(buffer)
	if err != nil {
		log.Println(err)
		return returnError(err.Error())
	} else {
		return returnSuccess("")
	}
}

func (fs *WebFS) Upload(parentDirectory string) string {
	fmt.Println("upload to folder: " + parentDirectory)
	parentDirectory = fs.normalizeInPath(parentDirectory)

	r := flaskgo.GetRequest()
	f, header, err := r.FormFile("uploaded_file")
	defer f.Close()

	if err != nil {
		log.Println(err)
		return returnError(err.Error())
	}
	var buffer = make([]byte, header.Size)
	_, err = f.Read(buffer)
	if err != nil {
		log.Println(err)
		return returnError(err.Error())
	}
	fullPath := path.Join(parentDirectory, header.Filename)
	out, err := os.Create(fullPath)
	defer out.Close()
	if err != nil {
		log.Println(err)
		return returnError(err.Error())
	}
	_, err = out.Write(buffer)
	if err != nil {
		log.Println(err)
		return returnError(err.Error())
	} else {
		return returnSuccess("")
	}
}

func initRoute(webFS *WebFS) {
	app := webFS.app
	app.AddRoute("/", app.Redirect("/fs/"))
	app.AddRoute("/fs", app.Redirect("/fs/"))
	app.AddRoute("/fs/", app.Redirect("/fs/list/"))
	app.AddRoute("/fs/list", app.Redirect("/fs/list/"))
	app.AddRoute("/fs/list/", webFS.Index)
	app.AddRoute("/fs/listup/@<path:path>", webFS.ListUp)
	app.AddRoute("/fs/list/@<path:path>", webFS.SimpleList)
	app.AddRoute("/fs/list/@<orderBy>/<boolean:isAsc>/<path:path>", webFS.List)
	app.AddRoute("/fs/mkdirs/@<path:path>", webFS.MakeDirs, "POST")
	app.AddRoute("/fs/delete/@<path:path>", webFS.Remove)
	app.AddRoute("/fs/download/@<path:path>", webFS.Download)
	app.AddRoute("/fs/view/@<path:path>", webFS.View)
	app.AddRoute("/fs/onedit/@<path:path>", webFS.OnEdit)
	app.AddRoute("/fs/save/@<path:path>", webFS.Save, "POST")
	app.AddRoute("/fs/upload/@<path:path>", webFS.Upload, "POST")
}

var port = 5007
var prefix = ""
var isService = false
var dir = "~"

func init() {
	flag.IntVar(&port, "port", port, "Listen Port")
	flag.StringVar(&prefix, "prefix", prefix, "Web URL prefix")
	flag.BoolVar(&isService, "service", isService, "Run as service")
	flag.StringVar(&dir, "dir", dir, "First dir")
	flag.Parse()
}

func main() {
	app := flaskgo.CreateAppWithPrefix(prefix)
	webFS := WebFS{&app}
	initRoute(&webFS)

	fmt.Println("Listen on port:" + strconv.Itoa(port))
	running := func() {
		app.Run(":" + strconv.Itoa(port))
	}
	if isService && runtime.GOOS == "windows" {
		runningInService(running)
	} else {
		running()
	}
}
