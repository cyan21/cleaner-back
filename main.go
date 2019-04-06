package main 
import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"encoding/json"
        "github.com/gorilla/mux"
	"github.com/cyan21/versioning"
        "net/http"
	"io/ioutil"
	"strconv"
	"strings"
)

type Versioning = versioning.Versioning
type SemVer20 = versioning.SemVer20

type Propertie struct {
  Key string
  Value string
}

type Result struct {
  Path string
  Properties []Propertie
}

type AQLResult struct {
  Results []Result
}

func (r Result) getPropValue (propName string) string {

  var propVal string

  for _, r := range r.Properties {
     if r.Key == propName {  
       propVal = r.Value 
     }
  }
  
  if propVal == "" {
    log.Fatal("Couldn't find package name")
  }

  return propVal  
}

/////// SUPPORT

func stringToInt(s string) int {
  i, err := strconv.Atoi(s) 

  if err != nil {
    log.Fatal("[Conversion] coulnd't extract ", s)
  }

  return i
}

func getSemVerFields(semver string) (int, int, int, string) {

  // extract semver fields
  tmp := strings.Split(semver, ".")

  maj := stringToInt(tmp[0])
  min := stringToInt(tmp[1])
  pat := stringToInt(strings.Split(tmp[2], "-")[0])
  mat := ""

  if len(strings.Split(tmp[2], "-")) > 1 {
    mat = strings.Split(tmp[2], "-")[1]
  } 
  
  return maj, min, pat, mat 
}

func allChecked(arr []bool) bool {
  for _, v := range arr { 
    if !v { return false }
  } 
  return true
}


//////////////////////////////:

func execAQL(url string, filename string, login string, pass string) []byte {

    pwd, _ := os.Getwd()
    file, err := os.Open(pwd + "/" + filename)

    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()


    request, err := http.NewRequest("POST", url, file)

    if err != nil {
        log.Fatal(err)
    }

    request.Header.Set("Content-Type", "text/plain")
    request.SetBasicAuth(login, pass)

    client := &http.Client{}

    response, err := client.Do(request)

    if err != nil {
        log.Fatal(err)
    }
    defer response.Body.Close()

    content, err := ioutil.ReadAll(response.Body)

    if err != nil {
        log.Fatal(err)
    }

    return content

}

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

func genFileSpec(repo string, pkgList map[string]map[string][]Versioning ) {

  fmt.Println("[genFileSpec] begin ...")

  filePattern := "{\"files\":[" 
    
  for pkgName, versList := range pkgList {

    exclVers := ""

    fmt.Println(pkgName)
    fmt.Println(versList)
    
    if len(versList["keep"]) > 0 {

      // loop in versions to keep/exclude 
      for i:= 0; i < len(versList["keep"]); i++ {
          exclVers += "\"*" + versList["keep"][i].Print() + "*\"," 
      }
     
      // add file pattern and remove last coma for excluded versions list
      filePattern += "{\"pattern\":\"" + repo + "/" + pkgName + "/*.tgz\",\"excludePatterns\": [" + exclVers[:len(exclVers) - 1]  + "]},"

    } // end if

  } // end for

  fmt.Println(filePattern[:len(filePattern)-1] + "]}")

  d1 := []byte(filePattern[:len(filePattern)-1] + "]}")
  err := ioutil.WriteFile("/tmp/delete.filespec", d1, 0644)

  if err != nil {
        panic(err)
  }

}

func parse2(toParse string, tagName string, tagVersion string, verstype string) map[string][]Versioning {

  //var arrRes []Result
  var arrRes AQLResult 
  var pkgList = make(map[string][]Versioning)

  err1 := json.Unmarshal([]byte(toParse), &arrRes)

  if err1 != nil {
    log.Fatal("Issue while unmarshalling")
  } 
  
  for _,res  := range arrRes.Results {
//   fmt.Println(res.Path) 

    pkgVersion := res.getPropValue(tagVersion)
    pkgName := res.getPropValue(tagName)

    if pkgList[pkgName] == nil {
      pkgList[pkgName] = make([]Versioning, 0, len(arrRes.Results))
    }  

    maj, min, pat, mat := getSemVerFields(pkgVersion)
    switch verstype {
      case "npm" : pkgList[pkgName] = append(pkgList[pkgName], SemVer20{Major: maj, Minor: min, Patch: pat, Maturity: mat})
      case "docker" : pkgList[pkgName] = append(pkgList[pkgName], SemVer20{Major: maj, Minor: min, Patch: pat, Maturity: mat})
      default: log.Fatal("[Parse2] unknow version type")
    }
  } // end for 

  return pkgList 
}


func parse(toParse []byte, tagName string, tagVersion string, verstype string) map[string][]Versioning {

  var arrRes AQLResult
  var pkgList = make(map[string][]Versioning)

  err1 := json.Unmarshal(toParse, &arrRes)

  if err1 != nil {
    log.Fatal("Issue while unmarshalling")
  } 
  
  for _,res  := range arrRes.Results {

    pkgVersion := res.getPropValue(tagVersion)
    pkgName := res.getPropValue(tagName)

    if pkgList[pkgName] == nil {
      pkgList[pkgName] = make([]Versioning, 0, len(arrRes.Results))
    }  

// use a constructor instead !!!!!
    maj, min, pat, mat := getSemVerFields(pkgVersion)

    switch verstype {
      case "npm" : pkgList[pkgName] = append(pkgList[pkgName], SemVer20{Major: maj, Minor: min, Patch: pat, Maturity: mat})
      case "docker" : pkgList[pkgName] = append(pkgList[pkgName], SemVer20{Major: maj, Minor: min, Patch: pat, Maturity: mat})
      default: log.Fatal("[Parse] unknow version type")
    }

  } // end for 

  return pkgList 
}


func sortVersion(toSort map[string][]Versioning) map[string][]Versioning {

  // to return
  sortedPkgs :=  make(map[string][]Versioning)
  
  // loop over package names
  for pkgName, versions := range toSort {
    
    sortedPkgs[pkgName] = make([]Versioning, 0, cap(versions))

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
            if versions[newestIndice].Newer(versions[j]) == 0 { 
              newestIndice = j 
            } 
          } 
        }
        
        sortedPkgs[pkgName] = append(sortedPkgs[pkgName], versions[newestIndice])
        usedIndices[newestIndice] = true
      } else {
        i++
      }// end if
 
    } // end loop versions
    

  } // end loop package name

  return sortedPkgs
}

func genAnswer(pkgsList map[string][]Versioning, limit int) map[string]map[string][]Versioning {

  var nLatest []Versioning 
  toSend := make(map[string]map[string][]Versioning)
  
  for k, v := range pkgsList {

    toSend[k] = make(map[string][]Versioning)

    if limit > len(v) {
      nLatest = v
      toSend[k]["delete"] = nil 
    } else {
      nLatest = v[:limit]
      toSend[k]["delete"] = v[limit:] 
    } 

    toSend[k]["keep"] = nLatest 
  }

  return toSend
}

func getArtifactList(w http.ResponseWriter, r *http.Request) {

  var pkgs map[string][]Versioning
  pkgType := mux.Vars(r)["type"]
  repoName := mux.Vars(r)["repo"]
  limit := stringToInt(mux.Vars(r)["nb"])

  var propName string 
  var propVers string
  var filename string 
  
  url := "http://192.168.41.41:8081/artifactory/api/search/aql"
  login := "admin" 
  pass := "password" 

  switch(pkgType) {
    case "docker": 
      filename = "list_docker.aql"
      propName = "docker.repoName"
      propVers = "docker.manifest"
    case "npm": 
      filename = "list_npm.aql"
      propName = "npm.name"
      propVers = "npm.version"
  }

  res := execAQL(url, filename, login, pass)
//  fmt.Println(string(res))

  pkgs = parse(res, propName, propVers, pkgType)
  sortedPkgs := sortVersion(pkgs)
//  fmt.Println(sortedPkgs)

  // for UI
  toSend := genAnswer(sortedPkgs,limit)
  
  // for deletion
  genFileSpec(repoName, toSend)

  json.NewEncoder(w).Encode(toSend)
}


func test(w http.ResponseWriter, r *http.Request) {

  var pkgs map[string][]Versioning
  limit := stringToInt(mux.Vars(r)["nb"])

/*
  url := "http://192.168.41.41:8081/artifactory/api/search/aql"
  filename := "list_npm.aql"
  login := "admin" 
  pass := "password" 
  fmt.Println("before AQL")
  res := execAQL(url, filename, login, pass)
  fmt.Println(string(res))
*/

  res := `{"results": [{"path":"/truc/chemin", "properties":[{"key": "npm.name", "value": "qotd"},{"key":"npm.version", "value": "0.0.1-alpha"}]},{"path":"/titi/batman", "properties":[{"key": "npm.name", "value": "cotd"},{"key":"npm.version","value":"0.12.0-alpha"}]},{"path":"/titi/batman", "properties":[{"key": "npm.name", "value": "qotd"},{"key":"npm.version", "value":"0.2.0-beta"}]}, {"path":"/truc/chemin2", "properties":[{"key": "npm.name", "value": "cotd"},{"key":"npm.version","value":"0.0.1-release"}]},{"path":"/truc/chemin2", "properties":[{"key": "npm.name", "value": "qotd"},{"key":"npm.version","value":"0.12.1-beta"}]}, {"path":"/truc/chemin2", "properties":[{"key": "npm.name", "value": "qotd"},{"key":"npm.version","value":"0.0.3-release"}]}]}`


  pkgs = parse2(res, "npm.name", "npm.version", "npm")
  fmt.Println("after parse", pkgs)	

  sortedPkgs := sortVersion(pkgs)
  fmt.Println(sortedPkgs)	

  json.NewEncoder(w).Encode(genAnswer(sortedPkgs,limit))

}


func main() {

  router := mux.NewRouter()
  router.HandleFunc("/{type}/{repo}/latest/{nb}", getArtifactList).Methods("GET")
  router.HandleFunc("/test/{nb}", test).Methods("GET")
  log.Fatal(http.ListenAndServe(":8000", router))
}
