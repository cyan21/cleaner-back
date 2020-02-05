package main 
import (
	"fmt"
	"log"
	"os"
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
  Repo string
  Properties []Propertie
}

type AQLResVers struct {
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

func sort(toParse []Result, propVers string, pkgType string) []Versioning {

  versions := make ([]Versioning, len(toParse))
 
  // get all versions
  for i:=0; i < len(toParse); i++ {

    // use a constructor instead !!!!!
    v := toParse[i].getPropValue(propVers)
    maj, min, pat, mat := getSemVerFields(v)

    switch pkgType {
      case "docker" : versions[i] = SemVer20{Major: maj, Minor: min, Patch: pat, Maturity: mat}
      default: log.Fatal("[Parse] unknow version type")
    }

  } // end for 

//  fmt.Println("versions :", versions)

  // sort array
  for i:=0; i<len(versions); i++ {
    for j:=i+1; j<len(versions); j++ {

//      fmt.Println("j:", versions[j],", i:", versions[i])

      if versions[j].Newer(versions[i]) == 1 { 
//        fmt.Println("OK permute !!")
        tmp := versions[i]
        versions[i] = versions[j]
        versions[j] = tmp 
      } 
    }
  } 

  return versions 
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

type SizeResult struct {
  Name string 
  Size int64
}

type AQLResSize struct {
  Results []SizeResult
}

func getTagSize(w http.ResponseWriter, r *http.Request) {

  var arrRes AQLResSize
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
//  var pkgs []Versioning
//  limit := stringToInt(mux.Vars(r)["nb"])

  var propName string 
  var propVers string
  
  switch(pkgType) {
    case "docker": 
      propName = "docker.repoName"
      propVers = "docker.manifest"
    case "npm": 
      propName = "npm.name"
      propVers = "npm.version"
  }

  buffer.WriteString("items.find({\"repo\":\"")
  buffer.WriteString(repoName)
  buffer.WriteString("\", \"name\":\"manifest.json\",\"path\": {\"$match\":\"")
  buffer.WriteString(img)
  buffer.WriteString("*\"}}).include(\"repo\",\"@")
  buffer.WriteString(propVers)
  buffer.WriteString("\", \"@")
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
  pkgs := sort(arrRes.Results, propVers, pkgType)
  fmt.Println(pkgs)

  // for UI
//  toSend := genAnswer(sortedPkgs,limit)
  json.NewEncoder(w).Encode(pkgs)

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
  router.HandleFunc("/docker/{repo}/{image}/{tag}/size", getTagSize).Methods("GET")
//  router.HandleFunc("/docker/{image}/size", getImageSize).Methods("GET")
//  router.HandleFunc("/docker/{repository}/size", getRepoSize).Methods("GET")
  log.Fatal(http.ListenAndServe(":8000", router))
}
