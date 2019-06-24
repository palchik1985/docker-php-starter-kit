package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/urfave/cli"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

// toDO create bash aliases: fill ~/.bash_aliases. Restart bash with "$ . ~/.bashrc"

var app = cli.NewApp()

func commands() {
	app.Commands = []cli.Command{
		//{
		//	Name:  "init", //toDo create init command for fill .env by command script
		//	Usage: "Write env variables if you don't have it",
		//	Action: func(c *cli.Context) {},
		//},
		{
			Name:    "create_project",
			Aliases: []string{"cp"},
			Usage:   "Creates Project. Creates file docker-compose.yml with needed containers which you choose in .env file. WARNING! Your changes in docker-compose.yml will be overrided!",
			Action: func(c *cli.Context) {
				// warning
				reader := bufio.NewReader(os.Stdin)
				// toDo: проверить есть ли файлы, и спрашивать только если они есть
				fmt.Print("Your changes in docker-compose.yml will be lost. Continue? (y/n) ")
				text, _ := reader.ReadString('\n')
				if string(text[0]) != "y" {
					fmt.Println("Aborted.")
					os.Exit(1)
				}

				err := godotenv.Load()
				if err != nil {
					log.Fatal("Error loading .env file")
				}

				// create config file
				var fileCompose bytes.Buffer

				// add start part to config file
				fileStart, err := ioutil.ReadFile("internal/config/start")
				if err != nil {
					log.Fatal("Error loading start part of file from ./internal/config/start. Aborted.")
				}
				fileCompose.Write(fileStart)
				fileCompose.Write([]byte("\n\n"))

				// add nginx_php

				fileNginx, err := ioutil.ReadFile("internal/config/nginx_php")
				if err != nil {
					log.Fatal("Error loading nginx file from ./internal/config/nginx_php. Aborted.")
				}
				fileCompose.Write(fileNginx)
				fileCompose.Write([]byte("\n\n"))

				// add config files for nginx_php
				_ = os.Mkdir("nginx_php", 0755)
				_ = os.Mkdir("nginx_php/config", 0755)

				file, _ := ioutil.ReadFile("internal/config/modules/trafex_php_nginx/Dockerfile")
				ioutil.WriteFile("nginx_php/Dockerfile", file, 0644)

				file, _ = ioutil.ReadFile("internal/config/modules/trafex_php_nginx/config/fpm-pool.conf")
				ioutil.WriteFile("nginx_php/config/fpm-pool.conf", file, 0644)

				file, _ = ioutil.ReadFile("internal/config/modules/trafex_php_nginx/config/nginx.conf")
				ioutil.WriteFile("nginx_php/config/nginx.conf", file, 0644)

				file, _ = ioutil.ReadFile("internal/config/modules/trafex_php_nginx/config/php.ini")
				ioutil.WriteFile("nginx_php/config/php.ini", file, 0644)

				file, _ = ioutil.ReadFile("internal/config/modules/trafex_php_nginx/config/supervisord.conf")
				ioutil.WriteFile("nginx_php/config/supervisord.conf", file, 0644)

				// Add framework/css nginx site config:
				// question to user
				reader = bufio.NewReader(os.Stdin)
				fmt.Print("\nPlease write needed framework/cms nginx config name, or write nothing for default. \n(Available list: ")
				files, _ := ioutil.ReadDir("./internal/config/modules/nginx/sites_conf")
				for _, filesite := range files {
					fmt.Print(filesite.Name() + ", ")
				}
				fmt.Print("):")
				// read answer
				text, _ = reader.ReadString('\n')

				// delete \n in the string's end
				text = strings.TrimSuffix(text, "\n")

				// read nginx site config file. Chosen file or default if didn't get any choose from user
				fileNginxConf := getfile(text)

				filestring := string(fileNginxConf)
				// vars replace in site.conf
				filestring = strings.Replace(filestring, "${APPNAME}", os.Getenv("APPNAME"), -1)
				filestring = strings.Replace(filestring, "${ENV}", os.Getenv("ENV"), -1)
				filestring = strings.Replace(filestring, "${SITE_WORKDIR_IN_CONTAINER}", os.Getenv("SITE_WORKDIR_IN_CONTAINER"), -1)
				// write site.conf
				ioutil.WriteFile("nginx_php/site.conf", []byte(filestring), 0644)

				// add db
				if strings.ToLower(os.Getenv("DB_DRIVER")) == "mysql" {
					fileMySql, err := ioutil.ReadFile("internal/config/mysql")
					if err != nil {
						log.Fatal(err)
					}
					fileCompose.Write(fileMySql)
				} else {
					// add file with the same name as lowercased DB_DRIVER value
					fileDb, err := ioutil.ReadFile(fmt.Sprintf("internal/config/%s", strings.ToLower(os.Getenv("DB_DRIVER"))))
					//fileDb, err = checkTabs(fileDb)
					if err != nil {
						log.Fatal(err)
					}
					fileCompose.Write(fileDb)
				}
				fileCompose.Write([]byte("\n\n"))

				// add other services which placed to OTHER_CONTAINERS ("space" delimitter. Example: OTHER_CONTAINERS=redis memcached phpmyadmin mailcatcher)
				// script will search files with same name (lowercase) as container names in OTHER_CONTAINERS
				if os.Getenv("OTHER_CONTAINERS") != "" {
					files := strings.Split(os.Getenv("OTHER_CONTAINERS"), " ")
					for _, file := range files {
						file, err := ioutil.ReadFile(fmt.Sprintf("internal/config/%s", strings.ToLower(file)))
						if err != nil {
							log.Fatal(err)
						}
						fileCompose.Write(file)
						fileCompose.Write([]byte("\n\n"))
					}
				}

				// save docker-compose.yml
				err1 := ioutil.WriteFile("docker-compose.yml", fileCompose.Bytes(), 0644)
				if err1 != nil {
					log.Fatal(err1)
				}
				fmt.Println("\nSuccess. \nSee docker-compose.yml for additional details.\nPHP config in \"php\" folder. Nginx config in \"nginx\" folder.")
			},
		},
		{
			Name:    "list_container",
			Aliases: []string{"ps"},
			Usage:   "Shows list of ALL runned containers",
			Action: func(c *cli.Context) {

				cmd := exec.Command("/bin/sh", "-c", "docker ps")
				cmd.Stdout = os.Stdout
				cmd.Run()
			},
		},
		{
			Name:    "composer_inst",
			Aliases: []string{"ci"},
			Usage:   "Usage: ci [path]. COMPOSER INSTALL. Default runs in PROJECT_ROOT. You can add another PATH with second argument. WARNING: Use only absolute path in container!",
			Action: func(c *cli.Context) {

				// get current user. It needed because files after composer install will be owned by root:root
				user, err := user.Current()
				if err != nil {
					panic(err)
				}
				// load .env
				err = godotenv.Load()
				if err != nil {
					log.Fatal("Error loading .env file")
				}
				if c.Args().First() == "" {
					cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec -u %s -i %s_php composer install", user.Name, os.Getenv("APPNAME")))
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.Run()
				} else {
					cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec -u %s -w %s -i %s_php composer install", user.Name, c.Args().First(), os.Getenv("APPNAME")))
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.Run()
				}
			},
		},
		{
			Name:    "command",
			Aliases: []string{"com"},
			Usage:   "Run command in container. Usage: com <container_name> \"<command>\". Few words command ONLY LIKE \"COMMAND NO ONE WORD\"! ",
			Action: func(c *cli.Context) {
				// load .env
				err := godotenv.Load()
				if err != nil {
					log.Fatal("Error loading .env file")
				}

				cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec -i %s_php %s", os.Getenv("APPNAME"), c.Args().First()))
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()
			},
		},
		{
			Name:    "stopall",
			Aliases: []string{"st"},
			Usage:   "Stops ALL runned Docker containers",
			Action: func(c *cli.Context) {

				exec.Command("/bin/sh", "-c", "docker stop $(docker ps -aq)").Run()

				fmt.Println("All containers stopped.")
				fmt.Println("DOCKER PS:")
				cmd := exec.Command("/bin/sh", "-c", "docker ps")
				cmd.Stdout = os.Stdout
				cmd.Run()
			},
		},
		{
			Name:    "logs",
			Aliases: []string{"lg"},
			Usage:   "Shows nginx/php error logs for this project.",
			Action: func(c *cli.Context) {
				// load .env
				err := godotenv.Load()
				if err != nil {
					log.Fatal("Error loading .env file")
				}
				cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker logs -f %s_nginx_php", os.Getenv("APPNAME")))
				cmd.Stdout = os.Stdout
				cmd.Run()
				fmt.Print("\n")
			},
		},
		{
			Name:    "dump_upload",
			Aliases: []string{"du"},
			Usage:   "Dump Upload. Uploads sql dump to mysql container. Place your dump.sql file to ./database folder before running",
			Action: func(c *cli.Context) {
				// load .env
				err := godotenv.Load()
				if err != nil {
					log.Fatal("Error loading .env file")
				}
				cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("cat ./database/dump.sql | docker exec -i %s_mysql /usr/bin/mysql -u %s --password=%s %s", os.Getenv("APPNAME"), os.Getenv("SQL_USER"), os.Getenv("SQL_USER"), os.Getenv("APPNAME")))
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()
			},
		},
		{
			Name:  "up",
			Usage: "docker-compose up -d. Runs all configured containers for this project.",
			Action: func(c *cli.Context) {
				// load .env
				err := godotenv.Load()
				if err != nil {
					log.Fatal("Error loading .env file")
				}
				cmd := exec.Command("/bin/sh", "-c", "docker-compose up -d --build")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()
			},
		},
		{
			Name:    "status",
			Aliases: []string{"s"},
			Usage:   "Statistics about all running docker containers",
			Action: func(c *cli.Context) {

				cmd := exec.Command("/bin/sh", "-c", "docker stats")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()
			},
		},
		{
			Name:    "disk",
			Aliases: []string{"d"},
			Usage:   "Statistics about disk usage by docker",
			Action: func(c *cli.Context) {

				cmd := exec.Command("/bin/sh", "-c", "docker system df")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()
			},
		},
		{
			Name:    "detstat",
			Aliases: []string{"ds"},
			Usage:   "Detail statistics about all docker containers, images, volumes on host machine",
			Action: func(c *cli.Context) {

				cmd := exec.Command("/bin/sh", "-c", "docker system df -v")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()
			},
		},
		{
			Name:  "resetall",
			Usage: "Resets all docker files to start state. If you want to delete mysql database files, you have run this script with SUDO rights!!!",
			Action: func(c *cli.Context) {
				// load .env
				err := godotenv.Load()
				if err != nil {
					log.Fatal("Error loading .env file")
				}
				fmt.Println("Stop all containers")
				cmd := exec.Command("/bin/sh", "-c", "docker stop $(docker ps -aq)")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()

				cmd = exec.Command("/bin/sh", "-c", "rm -vrf ./database ./nginx_php ./docker-compose.yml")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()
			},
		},
	}
}

func main() {
	// good PHP Dockerfile (alpine based) for Symfony https://github.com/eko/docker-symfony
	// todo: сделать проверку отступов в конфигах контейнеров: (split на строки и проверка первых символов)

	// override --help message
	cli.AppHelpTemplate = `
USAGE:
	mainscript (Rename filename to short name for fast usage!) <command>
{{if .Commands}}
COMMANDS:
{{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}
`

	commands()

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func getfile(text string) string {
	if text == "" {
		file, err := ioutil.ReadFile("internal/config/modules/nginx/site.conf")
		if err != nil {
			log.Fatal(fmt.Sprintf("Error loading file with entered name: %s", text))
		}
		return string(file)
	} else {

		path := fmt.Sprintf("internal/config/modules/nginx/sites_conf/%s", text)
		file, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatal(fmt.Sprintf("Error loading file with entered name: %s", text))
		}
		return string(file)
	}
}
