package webapp

//
//import (
//	"fmt"
//	"gopkg.in/yaml.v3"
//	"log"
//	"os"
//	"path"
//)
//
//type Titles map[string]map[string]string
//
//var Pagetitles Titles
//
//func ReadPagetitles() error {
//	filename := path.Join(Config.DataDirectory, "pagetitles.yaml")
//	file, err := os.ReadFile(filename)
//	if err != nil {
//		Logf(ErrorLevel, "%s\n", err)
//		Logf(ErrorLevel, "Unable to open Pagetitles file for reading.\nUsing default page titles\n")
//		Pagetitles = Titles{
//			"NewUser": {
//				"en": "Add User",
//				"de": "Benutzer anlegen",
//			},
//			"EditUser": {
//				"en": "Edit User",
//				"de": "Benutzer bearbeiten",
//			},
//		}
//		Pagetitles.Save(filename)
//		return err
//	}
//
//	err = yaml.Unmarshal(file, &Pagetitles)
//	if err != nil {
//		fmt.Println(err.Error())
//		return err
//	}
//
//	return nil
//}
//
//func (c *Titles) Yaml() []byte {
//	y, err := yaml.Marshal(c)
//	if err != nil {
//		Logf(ErrorLevel, "%s\n", err)
//		return nil
//	}
//	return y
//}
//
//func (c *Titles) Save(path string) {
//	err := os.WriteFile(path, c.Yaml(), 0600)
//	if err != nil {
//		Logf(ErrorLevel, "Unable to save page titles to file %s.\n%s\n", path, err)
//	}
//}
