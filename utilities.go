package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

// conf struct
type conf struct {
	Path           []string `yaml:"path"`
	Backupsize     []int    `yaml:"backupsize"`
	Workers        []int    `yaml:"Workers"`
	Files          []int    `yaml:"files"`
	Folders        []int    `yaml:"folders"`
	Size           []int    `yaml:"size"`
	TreeName       []string `yaml:"treeName"`
	BackupfileName []string `yaml:"backupfileName"`
	//SetsOfSizeno []int    `yaml:"SetsOfSizeno"`
}

//Weighted struct for setting counter
type Weighted struct {
	counter int
}

//NewWeighted function to set counter
func (p *Params) NewWeighted() {
	p.wt.counter = p.noOfWorkers

}

//TryAcquire function to decrement counter
func (p *Params) TryAcquire(n int) bool {

	if p.wt.counter > 0 {
		p.wt.counter--
		return true
	}
	return false

}

//Release function to decrement counter
func (p *Params) Release(n int) {
	p.wt.counter++
}

//getConf function to read from conf.yaml file
func (c *conf) GetConf() *conf {

	yamlFile, err := ioutil.ReadFile("conf.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

//Job struct to store node to be processed by goroutines
type Job struct {
	nd  *Node
	prt *Node
}

//Node containg the required fields
type Node struct {
	Name     string
	Info     *Info
	Children []*Node
	childWG  *sync.WaitGroup
	countWG  int
}

//Tree containg the required fields
type Tree struct {
	Root *Node
}

//Info containg the required fields
type Info struct {
	Path        string
	FileCount   int64
	FolderCount int64
	Size        int64
	IsDir       bool
}

//To print the tree using bfs
func (rt *Node) printTree() {

	if rt == nil {
		return
	}
	queue := []*Node{}

	queue = append(queue, rt)

	capacity := len(queue)
	//fmt.Println(capacity)
	for capacity > 0 {

		dequeNode := queue[0]

		queue = queue[1:]

		if dequeNode == nil {
			break
		}

		fmt.Println("Printed: ", dequeNode.Name, "       FileCount: ", dequeNode.Info.FileCount, "      FolderCount: ", dequeNode.Info.FolderCount, "Size: ", dequeNode.Info.Size)

		i := 0

		for i < len(dequeNode.Children) {

			queue = append(queue, dequeNode.Children[i])
			i = i + 1
		}
		capacity = len(queue)

	}

}

//Newtree creates initial root node
func Newtree(Root string) *Tree {
	tree := &Tree{}
	var child sync.WaitGroup
	tree.Root = &Node{countWG: 0, Info: &Info{Path: Root, FileCount: 0, FolderCount: 0, Size: 0, IsDir: true}, Name: Root, childWG: &child}
	return tree
}

//Details struct to store details of each node
type Details struct {
	totFiles   int64
	totFolders int64
	totSize    int64
}

/*func (d *Tree) scan() {

	createTree(d.Root)
}*/

//IsAnyGoroutine checks for empty goroutine
func (p *Params) IsAnyGoroutine() bool {
	if p.wt.counter > 0 {
		return true
	}
	return false

}

//CreateNewFolderNode creates folder node
func CreateNewFolderNode(fullPath string) *Node {

	var child sync.WaitGroup

	foldernode := &Node{Name: fullPath, Info: &Info{Path: fullPath, FileCount: 0, FolderCount: 0, Size: 0, IsDir: true}, childWG: &child, countWG: 0}
	return foldernode

}

//CreateNewFileNode creates file node
func CreateNewFileNode(fullPath string, filesize int64) *Node {

	var child sync.WaitGroup
	filenode := &Node{Name: fullPath, Info: &Info{Path: fullPath, FileCount: 0, FolderCount: 0, Size: filesize, IsDir: false}, childWG: &child}
	return filenode
}

//IsAnyJob checks for any job in channel
func IsAnyJob(chk bool) bool {
	if chk {
		return true
	}
	return false

}

//CalculateDetails calculates all info fields
func CalculateDetails(root *Node) (int64, int64, int64) {

	totalFiles, totalFolders, totalSize := int64(0), int64(0), int64(0)

	if root == nil || root.Name == "" {
		fmt.Println("Nothing to calculate")
	}

	totalFiles = calculateFiles(root)

	/*if err != nil {
		fmt.Println("error")
	}*/

	totalFolders = calculateDirectories(root)

	/*if err != nil {
		fmt.Println("error")
	}*/

	for i := 0; i < len(root.Children); i++ {
		totalFiles = totalFiles + root.Children[i].Info.FileCount
		totalFolders = totalFolders + root.Children[i].Info.FolderCount
		totalSize = totalSize + root.Children[i].Info.Size
	}

	return totalFiles, totalFolders, totalSize

}

/*CreateTree function
id = id of a goroutine
gos = status of goroutine(true=goroutine call,false=recusive call)
root if not nil its a recursive call with the root node provided
root if nil is a goroutine call and the node should be picked from job channel*/
func (p *Params) CreateTree(id int, gos bool, root *Node) {
	// to keep track from where the root node came
	var parent *Node
	parent = nil
	//indefinite for loop for goroutines to contineously listen to the job channel
	for {
		//its from goroutine
		if root == nil {

			select {
			case x, ok := <-p.jobs: //checks if any job is present on jobs channel
				if IsAnyJob(ok) { //found job

					got := p.TryAcquire(1) //trying to acquire a counter
					if got {               //got counter can further proceed with the job

						root = x.nd
						parent = x.prt
					}

				} else {
					//if jobs channel is closed its a signal that all jobs are done.
					//fmt.Println("Channel closed!")
					return
				}

			}
		}
		//reading the inside of root directories
		files, err := ioutil.ReadDir(root.Name)
		if err != nil {
			fmt.Println(err)
		}

		/*for entry in os.scandir(root.Name)
		  if not entry.name.startswith('.') and entry.is_dir():
		      yield entry.name*/

		//accessing all the files and directories

		for _, f := range files {

			fullPath := root.Name + "/" + f.Name()

			fullPath = strings.Replace(fullPath, "//", "/", -1)

			/*if given path is of a directory create Node and if any goroutine is empty put the node in channel or else
			call recursive */
			if info, err := os.Stat(fullPath); err == nil && info.IsDir() {

				childNode := CreateNewFolderNode(fullPath)
				root.Children = append(root.Children, childNode)

				//if any goroutine is empty
				if p.IsAnyGoroutine() {

					root.childWG.Add(1)
					p.jobs <- Job{childNode, root}

				} else { //all goroutines are busy do it by urself

					p.CreateTree(id, false, childNode)

				}

			} else { //if given path is of file just create a node and add it to the tree

				/*info, err := os.Stat(fullPath)
				if err != nil {
					log.Fatal(err)
				}*/

				root.Children = append(root.Children, CreateNewFileNode(fullPath, info.Size()))

			}

		}
		//wait for all Children who are getting processed by some goroutine
		root.childWG.Wait()

		//calculate total files, folders, total size of the current root(folder)
		root.Info.FileCount, root.Info.FolderCount, root.Info.Size = CalculateDetails(root)

		/*if its a recursive call and satisfy the condition then return
		gos=false as its not a goroutine
		parent=nil as it itself called*/
		if gos == false && parent == nil {
			return
		}
		/*end condition where it checks for the main root completely processed or not*/
		if root.Name == p.rootName && parent == nil {
			close(p.jobs)
			return
		}
		//if its a goroutine call
		if gos {
			//goroutine came from a parent just release the counter
			if parent != nil {
				parent.childWG.Done()
				p.Release(1)

			} else {
				return
			}

		}

		root = nil
		parent = nil

	}

}

/*
//To calculate total no of sub-directories in the given directory path
func calculateDirectories(root *Node) (int64, error) {

	if !root.Info.IsDir {
		return 0, nil
	}
	i := int64(0)
	files, err := ioutil.ReadDir(root.Name)
	if err != nil {

		return 0, err
	}
	for _, file := range files {
		if file.IsDir() {
			i++
		}
	}
	return i, nil

}*/
/*
//To calculate total no of files in the given directory path
func calculateFiles(root *Node) (int64, error) {

	if !root.Info.IsDir {
		return 0, nil
	}
	i := int64(0)
	files, err := ioutil.ReadDir(root.Name)
	if err != nil {
		return 0, err
	}
	for _, file := range files {
		if !file.IsDir() {
			i++
		}
	}
	return i, nil

}*/

//To calculate total no of files in the given directory path
func calculateFiles(root *Node) int64 {

	if !root.Info.IsDir {
		return 0
	}
	cnt := int64(0)
	for i := 0; i < len(root.Children); i++ {
		if !root.Children[i].Info.IsDir {
			cnt++
		}
	}
	return cnt

}

//PrintTreeInJSON function
func (d *Tree) PrintTreeInJSON(Rootpath string, bkpsize int64, treefileName string, backupfileName string) {

	//bytes, err := json.Marshal(d.Root)
	_, err := json.Marshal(d.Root)
	if err != nil {
		fmt.Println("failed to print tree")
	}

	//fmt.Println(string(bytes))

	file, _ := json.MarshalIndent(d.Root, "", "")
	_ = ioutil.WriteFile(treefileName, file, 0644)

	/*estimationdata := []byte("const treeData = ")
	estimationdata = append(estimationdata, file...)*/
	size := int64(1.5 * GB)
	bkpsets := BackupCreation2(d.Root, size)
	file, _ = json.MarshalIndent(bkpsets, "", "")
	_ = ioutil.WriteFile(backupfileName, file, 0644)

	/*estimationdata = append(estimationdata, []byte(";const suggestions = ")...)
	estimationdata = append(estimationdata, file...)

	_ = ioutil.WriteFile("../dataest/ui/data.js", estimationdata, 0644)

	/*file, _ = json.MarshalIndent(ltreligible, "", "")
	  _ = ioutil.WriteFile("ltreligible.json", file, 0644)*/

}

//To calculate total no of sub-directories in the given directory path
func calculateDirectories(root *Node) int64 {

	if !root.Info.IsDir {
		return 0
	}
	cnt := int64(0)

	for i := 0; i < len(root.Children); i++ {

		if root.Children[i].Info.IsDir {
			cnt++
		}

	}

	return cnt

}

//CreateWorkerPool function
func (p *Params) CreateWorkerPool() {

	p.glb.Add(p.noOfWorkers)
	for i := 0; i < p.noOfWorkers; i++ {

		go func(i int) {

			p.CreateTree(i, true, nil)
			p.glb.Done()

		}(i)

	}
	p.glb.Wait()

	return

}

//SetsOfSize containg the paths and the size
type SetsOfSize struct {
	Paths []string
	Size  int64
}

//SetsOfFileCount containg the paths and the size
type SetsOfFileCount struct {
	Paths []string
	Count int64
}

//Countset containing the size and path
type Countset struct {
	FileCount int64
	Path      string
}

//Countsets object
type Countsets []Countset

//Sizeset containing the size and path
type Sizeset struct {
	Size int64
	Path string
}

//Sizesets object
type Sizesets []Sizeset

/*SetsOfSize creation on file counts*/

func f1(root *Node, filecnt int64) []SetsOfFileCount {

	bkpsets := make([]SetsOfFileCount, 0)

	if filecnt == 0 {

		return bkpsets
	}

	if root.Info.FileCount <= filecnt {
		bkpsets = append(bkpsets, SetsOfFileCount{
			Paths: []string{root.Name},
			Count: root.Info.FileCount,
		})
		return bkpsets
	}

	dirs := make([]string, 0)
	queue := []*Node{}
	queue = append(queue, root)

	ss := make(Countsets, 0)

	for len(queue) > 0 {

		dequeNode := *queue[0]

		queue = queue[1:]

		if dequeNode.Info.FileCount > filecnt {

			//hasDir := bool(false)

			//Checking weather a child's path is of directory or a file
			/*for i := 0; i < len(dequeNode.Children); i++ {
				if dequeNode.Children[i].Info.IsDir {
					hasDir = true
				}

			}

			//If all the Children are files
			if !hasDir {
				dirs = append(dirs, dequeNode.Name)
				bkpsets = append(bkpsets, SetsOfSize{
					Paths: dirs,
					Size:  dequeNode.Info.Size,
				})
				dirs = nil

			}*/

			for i := 0; i < len(dequeNode.Children); i++ {

				/*if cnt == filecnt {

					bkpsets = append(bkpsets, SetsOfFileCount{
						Paths: dirs,
						Count: cnt,
					})
					cnt = 0
					dirs = nil

				}*/

				if !dequeNode.Children[i].Info.IsDir {

					ss = append(ss, Countset{
						Path:      dequeNode.Children[i].Name,
						FileCount: 1,
					})

				} else if dequeNode.Children[i].Info.IsDir { //If the child is a directory
					//If its filecount is greater than required filecnt
					if dequeNode.Children[i].Info.FileCount > filecnt {
						queue = append(queue, dequeNode.Children[i])

					} else { //If its filecount is less than required filecnt create sets
						ss = append(ss, Countset{
							Path:      dequeNode.Children[i].Name,
							FileCount: dequeNode.Children[i].Info.FileCount,
						})

					}
				}

			}
		} else {
			ss = append(ss, Countset{
				Path:      dequeNode.Name,
				FileCount: dequeNode.Info.FileCount,
			})

		}

	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].FileCount < ss[j].FileCount
	})

	curcnt := int64(0)
	i := 0
	j := len(ss) - 1

	//Creating SetsOfSizes of required size
	for i <= j {

		if i == j && curcnt+ss[j].FileCount <= filecnt {
			dirs = append(dirs, ss[j].Path)
			curcnt = curcnt + ss[j].FileCount
			i++
			j--
		} else if curcnt+ss[j].FileCount <= filecnt {
			dirs = append(dirs, ss[j].Path)
			curcnt = curcnt + ss[j].FileCount
			j--
		} else if curcnt+ss[i].FileCount <= filecnt {
			dirs = append(dirs, ss[i].Path)
			curcnt = curcnt + ss[i].FileCount
			i++
		} else {

			bkpsets = append(bkpsets, SetsOfFileCount{
				Paths: dirs,
				Count: curcnt,
			})
			dirs = nil
			curcnt = int64(0)

		}

	}

	//If any SetsOfSize left in dirs
	if dirs != nil && curcnt != int64(0) {
		bkpsets = append(bkpsets, SetsOfFileCount{
			Paths: dirs,
			Count: curcnt,
		})
		dirs = nil
		curcnt = int64(0)
	}

	return bkpsets

}

//BackupCreation2 function to create SetsOfSizes containing different subtrees
func BackupCreation2(root *Node, bkpsize int64) []SetsOfSize {
	bkpsets := make([]SetsOfSize, 0)

	if bkpsize == 0 {

		return bkpsets
	}

	rootflg := false //to check weather any directory present

	for i := 0; i < len(root.Children); i++ {
		if root.Children[i].Info.IsDir {
			rootflg = true
			break
		}
	}

	// if root dir has no further sub-directories
	if root.Children[0] == nil || !rootflg {
		bkpsets = append(bkpsets, SetsOfSize{
			Paths: []string{root.Name},
			Size:  root.Info.Size,
		})
		return bkpsets
	}

	dirs := make([]string, 0)
	queue := []*Node{}
	queue = append(queue, root)
	ss := make(Sizesets, 0)

	for len(queue) > 0 {

		dequeNode := *queue[0]

		queue = queue[1:]

		if dequeNode.Info.Size > bkpsize {

			hasDir := bool(false)

			//Checking weather a child's path is of directory or a file
			for i := 0; i < len(dequeNode.Children); i++ {
				if dequeNode.Children[i].Info.IsDir {
					hasDir = true
				}

			}

			//If all the Children are files
			if !hasDir {
				dirs = append(dirs, dequeNode.Name)
				bkpsets = append(bkpsets, SetsOfSize{
					Paths: dirs,
					Size:  dequeNode.Info.Size,
				})
				dirs = nil

			}

			for i := 0; i < len(dequeNode.Children); i++ {
				//If the child is a directory
				if dequeNode.Children[i].Info.IsDir {
					//If its size is greater than required bkpsize
					if dequeNode.Children[i].Info.Size > bkpsize {
						queue = append(queue, dequeNode.Children[i])

					} else { //If its size is less than required bkpsize create sets of there size and path

						ss = append(ss, Sizeset{
							Path: dequeNode.Children[i].Name,
							Size: dequeNode.Children[i].Info.Size,
						})
					}
				}

			}
		} else {
			ss = append(ss, Sizeset{
				Path: dequeNode.Name,
				Size: dequeNode.Info.Size,
			})

		}

	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Size < ss[j].Size
	})

	curSize := int64(0)
	i := 0
	j := len(ss) - 1

	//Creating SetsOfSizes of required size
	for i <= j {

		if i == j && curSize+ss[j].Size <= bkpsize {
			dirs = append(dirs, ss[j].Path)
			curSize = curSize + ss[j].Size
			i++
			j--
		} else if curSize+ss[j].Size <= bkpsize {
			dirs = append(dirs, ss[j].Path)
			curSize = curSize + ss[j].Size
			j--
		} else if curSize+ss[i].Size <= bkpsize {
			dirs = append(dirs, ss[i].Path)
			curSize = curSize + ss[i].Size
			i++
		} else {

			bkpsets = append(bkpsets, SetsOfSize{
				Paths: dirs,
				Size:  curSize,
			})
			dirs = nil
			curSize = int64(0)

		}

	}

	//If any SetsOfSize left in dirs
	if dirs != nil && curSize != int64(0) {
		bkpsets = append(bkpsets, SetsOfSize{
			Paths: dirs,
			Size:  curSize,
		})
		dirs = nil
		curSize = int64(0)
	}

	return bkpsets

}
