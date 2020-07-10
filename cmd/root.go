/*
Copyright © 2020 gikoha

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ConvertSql <filename>",
	Short: "convert MySQL SQL lines into Oracle SQL",
	Long: `convert MySQL SQL dump (from TablePlus) into Oracle SQL (SQLdeveloper-compliant)
 and print it to standard output.
Beware your primary keys are ignored, so you have to make it manually.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires filename")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		analyze(args[0])
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ConvertSql.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".ConvertSql" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".ConvertSql")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// tableElementsChanger : replace string and print, use for private conversion
func tableElementsChanger(s string, notNullMap map[string]bool) {
	s = strings.TrimRight(s, ",")
	slice := strings.Split(s, " ")
	slice[0] = strings.ToUpper(strings.Replace(slice[0], "`", "\"", -1))
	s1 := strings.Join(slice, " ")

	s2 := strings.Replace(s1, "int", "NUMBER", 1)
	s1 = strings.Replace(s2, "float", "FLOAT(126)", 1)
	if strings.Contains(s1, "NOT NULL") {
		notNullMap[slice[0]] = true
		s1 = strings.Replace(s1, "NOT NULL", "", 1)
	}
	s1 = strings.Replace(s1, "CHARACTER SET utf8mb4", "", 1)
	s1 = strings.Replace(s1, "COLLATE utf8mb4_ja_0900_as_cs", "COLLATE \"USING_NLS_COMP\"", 1)
	s1 = strings.Replace(s1, "varchar", "NVARCHAR2", 1)
	s1 = strings.Replace(s1, "unsigned", "", 1)
	s1 = strings.Replace(s1, "AUTO_INCREMENT", "", 1)
	if strings.Contains(s1, "NVARCHAR2") {
		// default null was not accepted
		s1 = strings.Replace(s1, "DEFAULT NULL", "DEFAULT ''", 1)
	}
	fmt.Print(s1)
}

// Tables table attribute definition
type Tables struct {
	tableName  string
	rowNotNull map[string]bool
}

func analyze(fname string) {
	file, err := os.Open(fname)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(err)
			os.Exit(1)
		} else {
			panic(err)
		}
	}
	defer file.Close()

	sc := bufio.NewScanner(file)
	var tableslice []Tables

	for i := 1; sc.Scan(); i++ {
		if err := sc.Err(); err != nil {
			panic(err)
			// エラー処理
		}
		str := sc.Text()
		s := strings.TrimSpace(str)
		if strings.HasPrefix(s, "/*") || strings.HasPrefix(s, "--") || strings.HasPrefix(s, "//") {
			// comment line
			fmt.Println(s)
			continue
		}
		if strings.HasPrefix(s, "DROP TABLE") {
			// ignore
			continue
		}
		if strings.HasPrefix(s, "CREATE TABLE") {
			// CREATE TABLE
			slice := strings.Split(s, " ")
			slice[2] = strings.ToUpper(strings.Replace(slice[2], "`", "\"", 2))
			tableName := slice[2]
			tableRowNamesNotNull := make(map[string]bool)

			fmt.Println(strings.Join(slice, " "))
			firstcreate := false
			for ; sc.Scan(); i++ {
				s = strings.TrimSpace(sc.Text())
				if strings.HasPrefix(s, ")") {

					break
				}
				if strings.HasPrefix(s, "PRIMARY KEY") {
					continue
				}
				if firstcreate {
					fmt.Println(",")
				}
				firstcreate = true
				tableElementsChanger(s, tableRowNamesNotNull)
			}
			var t Tables
			t.tableName = tableName
			t.rowNotNull = tableRowNamesNotNull
			tableslice = append(tableslice, t)

			fmt.Println(
				`
)  DEFAULT COLLATION "USING_NLS_COMP" SEGMENT CREATION IMMEDIATE 
PCTFREE 10 PCTUSED 40 INITRANS 1 MAXTRANS 255 
NOCOMPRESS LOGGING
STORAGE(INITIAL 65536 NEXT 1048576 MINEXTENTS 1 MAXEXTENTS 2147483645
PCTINCREASE 0 FREELISTS 1 FREELIST GROUPS 1
BUFFER_POOL DEFAULT FLASH_CACHE DEFAULT CELL_FLASH_CACHE DEFAULT)
TABLESPACE "DATA" ;`)
			continue
		}

		if strings.HasPrefix(s, "INSERT INTO") {
			// INSERT conversion to oracle style
			fmt.Println("SET DEFINE OFF;") // oracle use "&" for special meaning

			s = strings.ToUpper(strings.Replace(s, "`", "\"", -1))
			for ; sc.Scan(); i++ {
				s2 := strings.TrimSpace(sc.Text())
				if strings.HasPrefix(s2, "(") {
					if strings.HasSuffix(s2, ",") {
						s2 = strings.TrimRight(s2, ",") // cut comma
					}
					if strings.HasSuffix(s2, ";") {
						s2 = strings.TrimRight(s2, ";") // cut semicolon
					}
					fmt.Println(s, " ", s2, ";")
					continue
				}
				break
			}
			// INSERT end. make primary key

			continue
		}

		fmt.Println(s)
	}

	// 各テーブルの NOT NULL 対応
	for _, t := range tableslice {

		for index, value := range t.rowNotNull {
			if value {
				fmt.Println("ALTER TABLE ", t.tableName, "MODIFY (", index, " NOT NULL ENABLE );")
			}
			delete(t.rowNotNull, index)
		}
	}

}
