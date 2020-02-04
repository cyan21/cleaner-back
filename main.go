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
  customlog "github.com/jfrog/jfrog-client-go/utils/log"
  "github.com/jfrog/jfrog-client-go/artifactory"
  "github.com/jfrog/jfrog-client-go/artifactory/services"
//  utils "github.com/jfrog/jfrog-client-go/artifactory/services/utils"
  "github.com/jfrog/jfrog-client-go/artifactory/auth"
  "bytes"
)

var rtManager artifactory.ArtifactoryServicesManager

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

///////////////////////////////////////////////
/////////////////// REFACTO //////////////////////
///////////////////////////////////////////////

type VersResult struct {
  Repo string 
  Properties []Propertie
}

type AQLResVers struct {
  Results []VersResult
}

type SizeResult struct {
  Name string 
  Size int64
}

type AQLRes struct {
  Results []SizeResult
}

func getTagSize(w http.ResponseWriter, r *http.Request) {

  var arrRes AQLRes
  var buffer bytes.Buffer
  var size int64 = 0
  repo := mux.Vars(r)["repo"]
  img := mux.Vars(r)["image"]
  tag := mux.Vars(r)["tag"]

  // run AQL to get all items size in folder (image tag)
  buffer.WriteString("items.find({\"repo\":\"")
  buffer.WriteString(repo)
  buffer.WriteString("\", \"path\":\"")
  buffer.WriteString(img)
  buffer.WriteString("/")
  buffer.WriteString(tag)
  buffer.WriteString("\"}).include(\"name\",\"size\")")

  fmt.Println(buffer.String())
  toParse, _ := rtManager.Aql(buffer.String()) 

  // extract and add size
  err1 := json.Unmarshal(toParse, &arrRes)

  if err1 != nil {
    log.Fatal(err1)
  } 
  for _,res := range arrRes.Results {
    size += res.Size
  } 
  
  m := make(map[string]string)
  m["Kb"] = strconv.FormatInt(size,10)
  m["Mb"] = fmt.Sprintf("%.2f", float64(size)/(1024*1024))
  m["Gb"] = fmt.Sprintf("%.2f", float64(size)/(1024*1024*1024))
  fmt.Println("size : ", m["Kb"], " KB, ", m["Mb"], " MB, ", m["Gb"], " GB")

  json.NewEncoder(w).Encode(m)
  
}

func getImageSize(w http.ResponseWriter, r *http.Request) {
  // get Artifactory Connection
  
  // call API to get all tags for an image
  
  // foreach tag => call getTagSize

  // return map where key = tag and value = size
}

func getRepoSize(w http.ResponseWriter, r *http.Request) {
  // get Artifactory Connection

  // call API to get all images in a repo
  
  // foreach image => call getImageSize

  // return map of map where key = image name and value = map returned by getImageSize 
}


func initService() artifactory.ArtifactoryServicesManager {


  // init log file
  file, _ := os.Create("./service.log")

  if os.Getenv("LOG_LEVEL") != "" {   
    switch os.Getenv("LOG_LEVEL") {
      case "ERROR": customlog.SetLogger(customlog.NewLogger(customlog.ERROR, file))
      default: customlog.SetLogger(customlog.NewLogger(customlog.INFO, file))
    }
  } else {
    customlog.SetLogger(customlog.NewLogger(customlog.INFO, file))
  }

  // check env variable for connection to Artifactory
  if os.Getenv("ART_URL") == "" || os.Getenv("ART_USER") == "" || os.Getenv("ART_PASS") == "" {  
   fmt.Println("ART_URL, ART_USER, ART_PASS are required environment variables !")
    os.Exit(2)
  } 

  // set up connection to Artifactory
  rtDetails := auth.NewArtifactoryDetails()
  rtDetails.SetUrl(os.Getenv("ART_URL"))
  rtDetails.SetUser(os.Getenv("ART_USER"))
  rtDetails.SetPassword(os.Getenv("ART_PASS"))

  serviceConfig, err := artifactory.NewConfigBuilder().
    SetArtDetails(rtDetails).
    SetDryRun(false).
    Build()

  if err != nil {
    log.Fatal("Init service config failed with url: ", os.Getenv("ART_URL"),", user: ",os.Getenv("ART_USER"))
  }   

  art, _ := artifactory.New(&rtDetails, serviceConfig)

  _, err = art.Ping()

  if err != nil {
    log.Fatal(err)
  }   
  
  return *art
}

///////////////////////////////////////////////
/////////////////// MAIN //////////////////////
///////////////////////////////////////////////

func getArtifactList(w http.ResponseWriter, r *http.Request) {

  var arrRes AQLResVers
  var buffer bytes.Buffer

  pkgType := mux.Vars(r)["type"]
  repoName := mux.Vars(r)["repo"]
  img := mux.Vars(r)["img"]
//  var pkgs map[string][]Versioning
//  limit := stringToInt(mux.Vars(r)["nb"])

  var propName string 
  var propVers string
  
  switch(pkgType) {
    case "docker": 
      propName = "@docker.repoName"
      propVers = "@docker.manifest"
    case "npm": 
      propName = "npm.name"
      propVers = "npm.version"
  }

  buffer.WriteString("items.find({\"repo\":\"")
  buffer.WriteString(repoName)
  buffer.WriteString("\", \"name\":\"manifest.json\",\"path\": {\"$match\":\"")
  buffer.WriteString(img)
  buffer.WriteString("*\"}}).include(\"repo\",\"")
  buffer.WriteString(propVers)
  buffer.WriteString("\", \"")
  buffer.WriteString(propName)
  buffer.WriteString("\")")

  fmt.Println("AQL: ", buffer.String())

  toParse, _ := rtManager.Aql(buffer.String()) 

  // extract and add size
  err1 := json.Unmarshal(toParse, &arrRes)

  if err1 != nil {
    log.Fatal(err1)
  } 
  fmt.Println(arrRes)
/*
  pkgs = parse(res, propName, propVers, pkgType)
  sortedPkgs := sortVersion(pkgs)
//  fmt.Println(sortedPkgs)

  // for UI
  toSend := genAnswer(sortedPkgs,limit)
  
  // for deletion
  genFileSpec(repoName, toSend)

  json.NewEncoder(w).Encode(toSend)
*/
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

  test := true 
  router := mux.NewRouter()
  rtManager = initService()

  if (test) {
    params := services.NewSearchParams()
    params.Pattern = "mygeneric-local/Carefirst.jpg"
    params.Recursive = true
    resultItems, _ := rtManager.SearchFiles(params)
    fmt.Println(resultItems)
  }
  
  router.HandleFunc("/{type}/{repo}/{img}/latest/{nb}", getArtifactList).Methods("GET")
//  router.HandleFunc("/test/{nb}", test).Methods("GET")
  router.HandleFunc("/docker/{repo}/{image}/{tag}/size", getTagSize).Methods("GET")
//  router.HandleFunc("/docker/{image}/size", getImageSize).Methods("GET")
//  router.HandleFunc("/docker/{repository}/size", getRepoSize).Methods("GET")
  log.Fatal(http.ListenAndServe(":8000", router))
}
