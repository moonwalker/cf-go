package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

var (
	dropSchema bool
)

func init() {
	schemaCmd.PersistentFlags().BoolVarP(&dropSchema, "drop", "d", false, "drop schema")
	schemaCmd.AddCommand(pgSchemaCmd)
}

var pgSchemaCmd = &cobra.Command{
	Use:   "pg",
	Short: "Creates postgres schema",

	Run: func(cmd *cobra.Command, args []string) {
		if len(schemaDatabaseURL) > 0 {
			log.Println("creating postgres schema...")
		}

		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:   apiURL,
			SpaceID:  SpaceId,
			CdnToken: CdnToken,
		})

		space, err := client.Spaces.GetSpace()
		if err != nil {
			log.Fatal(err)
		}

		types, err := client.ContentTypes.GetTypes()
		if err != nil {
			log.Fatal(err)
		}

		schema := gontentful.NewPGSQLSchema(schemaName, dropSchema, space, types.Items)
		str, err := schema.Render()
		if err != nil {
			log.Fatal(err)
		}

		if len(schemaDatabaseURL) == 0 {
			fmt.Println(str)
			return
		} else {
			log.Println("postgres schema successfully created")
		}

		log.Println("executing postgres schema...")
		if dropSchema {
			log.Println("existing schema will be dropped")
		}
		db, _ := sql.Open("postgres", schemaDatabaseURL)
		txn, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}

		_, err = db.Exec(str)
		if err != nil {
			log.Fatal(err)
		}

		err = txn.Commit()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("postgres schema successfully executed")
	},
}