package main

import (
	"fmt"
	"log"
	"os/exec"
	"encoding/json"
	"github.com/cyan21/versioning"
//	"io"
	"strconv"
	"strings"
)

type NPMSemVer = versioning.SemVer

func callCLI() string {


	path, err := exec.LookPath("jfrog")

	if err != nil {
		log.Fatal("installing fortune is in your future")
	} 

	fmt.Println("JFrog CLI found in" + path )
	
	output, err := exec.Command("jfrog", "rt", "search", "--spec=search.filespec").Output()

	if err != nil {
		log.Fatal("Issue while invoking jfrog CLI")
	} 
	
	//fmt.Println(string(output))	
	return 	string(output)

}

type Result struct {
  Path string
  Props interface{} 
}

func parse(toParse string, tagName string, tagVersion string) map[string][]NPMSemVer {


  var arrRes []Result

  var pkgList = make(map[string][]NPMSemVer)
  var tmp []string
  var name, version, mat string
  debug := false

  err1 := json.Unmarshal([]byte(toParse), &arrRes)

  if err1 != nil {
    log.Fatal("Issue while invoking jfrog CLI")
   } 
	
  for _,res  := range arrRes {
//		fmt.Printf("Type assert %s\n",res.Path )
    listProps := res.Props.(map[string]interface{})
//		fmt.Printf("List props : %s\n",listProps )

    if pkgName, ok := listProps[tagName]; ok { 
      name = extractValue(pkgName)
//		fmt.Println(tagName," : ", name, "Present?", ok)
    }

    if pkgVers, ok := listProps[tagVersion]; ok {
      version = extractValue(pkgVers)
//		fmt.Println(tagVersion, " : ", version, "Present?", ok)
    }
		
    // check if package name exists in map

    if val, ok := pkgList[name]; ok {
      // exists :  add version 

      if debug { 
	fmt.Println("pkgName ", name," already present")
        fmt.Println("val: ", val) 
      }

      tmp = strings.Split(version, ".")
//			fmt.Println(tmp)

      maj, err := strconv.Atoi(tmp[0])
      min, err2 := strconv.Atoi(tmp[1])
      pat, err3 := strconv.Atoi(strings.Split(tmp[2], "-")[0])
      mat = ""

      if len(strings.Split(tmp[2], "-")) > 1 {
        mat = strings.Split(tmp[2], "-")[1]
      }

      if (err == nil && err2 == nil && err3 == nil) {
        pkgList[name] = append(pkgList[name], NPMSemVer{Major: maj, Minor: min, Patch: pat, Maturity: mat})
      }
    } else {
	// doesn't exist : add map 
//			fmt.Println("New pkgName : ", name)
      tmp = strings.Split(version, ".")
// 			fmt.Println(tmp)

      pkgList[name] = make([]NPMSemVer, 0, len(arrRes))

      maj, err := strconv.Atoi(tmp[0])
      min, err2 := strconv.Atoi(tmp[1])
      pat, err3 := strconv.Atoi(strings.Split(tmp[2], "-")[0])
      mat = ""

      if len(strings.Split(tmp[2], "-")) > 1 {
        mat = strings.Split(tmp[2], "-")[1]
      }

      if (err == nil && err2 == nil && err3 == nil) {
        pkgList[name] = append(pkgList[name], NPMSemVer{Major: maj, Minor: min, Patch: pat, Maturity: mat})
      }
    } // enf if

  } // end for 

//  fmt.Println(pkgList)
  return pkgList
}


func extractValue (itf interface{}) string {
  
  res := "unknown" 

  switch v := itf.(type) {
    case []interface{}:
      res = v[0].(string)

    default:
      fmt.Println("type unknown") 
  }
  
  return res

}

func sortVersion(toSort map[string][]NPMSemVer) map[string][]NPMSemVer {

  // to return
  sortedPkgs :=  make(map[string][]NPMSemVer)
  
  // loop over package names
  for pkgName, versions := range toSort {
    
    sortedPkgs[pkgName] = make([]NPMSemVer, 0, cap(versions))

    // store indices pointing to NPMSemver already sorted out
    usedIndices := make([]bool, len(versions))
    i := 0

    // loop over package versions until all elements were sorted out
    for !allChecked(usedIndices) {
      
      if !usedIndices[i] {
        newestIndice := i 

        // look for newest version 
        for j := i + 1; j < len(versions); j++ { 

          if !usedIndices[newestIndice] && !usedIndices[j] {
/*
            fmt.Println("indice ", newestIndice, " not used")
            fmt.Println(versions[newestIndice], " Vs ", versions[j])
*/
            if versions[newestIndice].Newer(versions[j]) == 0 { 
//              fmt.Println(versions[j], " won")
              newestIndice = j 
            } 
          } 
/*else { 
            fmt.Println("indice ", newestIndice, " used") 
          }
*/
        }
        
        sortedPkgs[pkgName] = append(sortedPkgs[pkgName], versions[newestIndice])
        usedIndices[newestIndice] = true
/*
	fmt.Println("newestIndice : ", newestIndice, "; value : ",versions[newestIndice])
        fmt.Println(usedIndices)
*/        
      } else {
        i++
      }// end if
 
    } // end loop versions
    

  } // end loop package name

  return sortedPkgs
}

func printRes(res map[string][]NPMSemVer) {

  for k, v := range res {
    fmt.Println("package :", k) 
    fmt.Println("versions :", v) 
    
  }
}

func allChecked(arr []bool) bool {
  for _, v := range arr { 
    if !v { return false }
  } 
  return true
}


func main() {

  pkgType := "npm"
  var pkgs map[string][]NPMSemVer

//	res := callCLI()
	res := `[{"path":"/truc/chemin", "props":{"npm.version": ["0.0.1-alpha"], "npm.name": ["custom-picture"]}},{"path":"/titi/batman", "props":{"npm.version": ["0.12.0-alpha"], "npm.name": ["custom-picture"]}},{"path":"/titi/batman", "props":{"npm.version": ["0.2.0-beta"], "npm.name": ["custom-picture"]}}, {"path":"/truc/chemin2", "props":{"npm.version": ["0.0.1-release"], "npm.name": ["custom-picture"]}}, {"path":"/truc/chemin2", "props":{"npm.version": ["0.0.1-beta"], "npm.name": ["custom-gallery"]}}, {"path":"/truc/chemin2", "props":{"npm.version": ["0.0.1-release"], "npm.name": ["custom-gallery"]}}]`


  switch pkgType {
    case "npm" :
      // get package name and version from properties 
      pkgs = parse(res, "npm.name", "npm.version")
    case "maven" :
      // get package name and version from path 
      fmt.Println("Maven type")
    default : 
      fmt.Println("Unknown")
  } 

//fmt.Println(pkgs)	
  sortedPkgs := sortVersion(pkgs)
//fmt.Println(sortedPkgs)	
  for _, v := range sortedPkgs {
    nLatest := v[:3]
    fmt.Println(cap(nLatest))
    fmt.Println(nLatest)
  }
  
  // send JSON data  
  //sendJSONResponse()
}
