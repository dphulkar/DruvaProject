package main

import (
	"testing"
)

/*var glb sync.WaitGroup
var jobs = make(chan Job, 100000)
var rootName string
var noOfWorkers int
var wt Weighted*/

//TestCases test function
func TestCases(t *testing.T) {
	var c conf
	c.GetConf()
	var p Params
	for x := 0; x < len(c.Path); x++ {
		gtree := Newtree(c.Path[x])
		p.rootName = c.Path[x]
		p.noOfWorkers = 2

		//setting the counter
		p.NewWeighted()
		var j = make(chan Job, 100000)
		p.jobs = j
		//pushing root node to the channel
		p.jobs <- Job{gtree.Root, nil}

		//fmt.Println("Creation of tree starts....")
		p.CreateWorkerPool()
		//fmt.Println("Tree created successfully....")
		files, folders, size := gtree.Root.Info.FileCount, gtree.Root.Info.FolderCount, gtree.Root.Info.Size

		if files == int64(c.Files[x]) {
			t.Logf("Success,expected %v,got %v", c.Files[x], files)
		} else {
			t.Errorf("failed,expected %v,got %v", c.Files[x], files)
		}

		if folders == int64(c.Folders[x]) {
			t.Logf("Success,expected %v,got %v", c.Folders[x], folders)
		} else {
			t.Errorf("failed,expected %v,got %v", c.Folders[x], folders)
		}

		if size == int64(c.Size[x]) {
			t.Logf("Success,expected %v,got %v", c.Size[x], size)
		} else {
			t.Errorf("failed,expected %v,got %v", c.Size[x], size)
		}
	}

}
