package main

import (
	"github.com/iancoleman/strcase"
	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func main() {
	// Initialize the generator with configuration
	config := gen.Config{
		OutPath:       "./internal/model", // output directory, default value is ./query
		Mode:          gen.WithDefaultQuery | gen.WithQueryInterface,
		FieldNullable: true,
	}
	config.WithJSONTagNameStrategy(func(columnName string) (tagContent string) {
		return strcase.ToLowerCamel(columnName)
	})
	g := gen.NewGenerator(config)

	// Initialize a *gorm.DB instance
	db, _ := gorm.Open(mysql.Open("root:Fqx_998875@tcp(47.76.150.244:3306)/auto_go_db?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{})

	// Use the above `*gorm.DB` instance to initialize the generator,
	// which is required to generate structs from db when using `GenerateModel/GenerateModelAs`

	g.UseDB(db)
	g.GenerateModel("account_coin")
	// Execute the generator
	g.Execute()
}
