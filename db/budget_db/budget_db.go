package budget_db

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"os"

	"github.com/ibilalkayy/flow/db"
	"github.com/ibilalkayy/flow/internal/common/functions"
	"github.com/ibilalkayy/flow/internal/common/structs"
	"github.com/jedib0t/go-pretty/v6/table"
)

func CreateBudget(bv *structs.BudgetVariables, basePath string) error {
	data, err := db.Table(basePath, "001_create_budget_table.sql", 0)
	if err != nil {
		return err
	}

	query := "INSERT INTO Budget(categories, amounts, spent, remaining) VALUES($1, $2, $3, $4)"
	insert, err := data.Prepare(query)
	if err != nil {
		return err
	}
	defer insert.Close()

	if len(bv.Category) != 0 {
		_, err = insert.Exec(bv.Category, bv.Amount, 0, 0)
		if err != nil {
			return err
		}
		fmt.Println("Budget data is successfully inserted!")
	} else {
		return errors.New("category can't be empty")
	}
	return nil
}

func ViewBudget(category string) ([4]interface{}, error) {
	// Create a new instance of BudgetVariables to hold the retrieved data
	bv := new(structs.BudgetVariables)

	// Connect to the database
	db, err := db.Connection()
	if err != nil {
		return [4]interface{}{}, err
	}
	defer db.Close()

	// Prepare the table writer
	tw := table.NewWriter()
	tw.AppendHeader(table.Row{"Category", "Amount"})

	// Initialize total amount
	totalAmount := 0

	// Query the database based on the provided category
	var rows *sql.Rows
	if len(category) != 0 {
		query := "SELECT categories, amounts, spent FROM Budget WHERE categories=$1"
		rows, err = db.Query(query, category)
	} else {
		query := "SELECT categories, amounts, spent FROM Budget"
		rows, err = db.Query(query)
	}
	if err != nil {
		return [4]interface{}{}, err
	}
	defer rows.Close()

	// Iterate over the rows and add them to the table writer
	for rows.Next() {
		if err := rows.Scan(&bv.Category, &bv.Amount, &bv.Spent); err != nil {
			return [4]interface{}{}, err
		}
		// Check if amount is empty
		if bv.Amount != 0 {
			tw.AppendRow([]interface{}{bv.Category, bv.Amount})
			totalAmount += bv.Amount
		}
	}

	// Add total amount row to the table
	tw.AppendFooter(table.Row{"Total Amount", totalAmount})

	// Render the table
	tableRender := "Budget Data\n" + tw.Render()

	details := [4]interface{}{tableRender, bv.Category, bv.Amount, bv.Spent}
	return details, nil
}

func RemoveBudget(category string) error {
	db, err := db.Connection()
	if err != nil {
		return err
	}

	query := "DELETE FROM Budget WHERE categories=$1"
	remove, err := db.Prepare(query)
	if err != nil {
		return err
	}

	defer remove.Close()

	if len(category) != 0 {
		_, err = remove.Exec(category)
		if err != nil {
			return err
		}
		fmt.Printf("'%s' category is successfully removed!\n", category)
	} else {
		fmt.Println("First enter the category and then remove it")
	}
	return nil
}

func UpdateBudget(old, new string, amount, spent, remaining int) error {
	var count int
	var query string
	var params []interface{}

	db, err := db.Connection()
	if err != nil {
		return err
	}

	// Check if the old category exists
	err = db.QueryRow("SELECT COUNT(*) FROM Budget WHERE categories = $1", old).Scan(&count)
	if err != nil {
		return err
	}

	// If the old category does not exist, return an error
	if count == 0 {
		return errors.New("'" + old + "'" + " category does not exist")
	}

	if len(new) != 0 && amount != 0 {
		query = "UPDATE Budget SET categories=$1, amounts=$2 WHERE categories=$3"
		params = []interface{}{new, amount, old}
	} else if len(new) != 0 {
		query = "UPDATE Budget SET categories=$1 WHERE categories=$2"
		params = []interface{}{new, old}
	} else if amount != 0 {
		query = "UPDATE Budget SET amounts=$1 WHERE categories=$2"
		params = []interface{}{amount, old}
	} else if spent != 0 {
		query = "UPDATE Budget SET spent=$1, remaining=$2 WHERE categories=$3"
		params = []interface{}{spent, remaining, old}
	} else {
		fmt.Println("No field provided to adjust")
	}

	_, err = db.Exec(query, params...)
	if err != nil {
		return err
	}
	return nil
}

func GetBudgetData(filepath, filename string) error {
	bv := new(structs.BudgetVariables)
	db, err := db.Connection()
	if err != nil {
		return err
	}

	query := "SELECT categories, amounts FROM Budget"
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	file, err := os.Create(filepath + "/" + filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"Category", "Amount"}
	if err := writer.Write(header); err != nil {
		return err
	}
	for rows.Next() {
		if err := rows.Scan(&bv.Category, &bv.Amount); err != nil {
			return err
		}

		var data []string
		amountStr := functions.IntToString(bv.Amount)
		data = append(data, bv.Category, amountStr)
		if err := writer.Write(data); err != nil {
			return err
		}

		if err := rows.Err(); err != nil {
			return err
		}
	}
	fmt.Printf("Successfully created a '%s' file in '%s'\n", filename, filepath)
	return nil
}
