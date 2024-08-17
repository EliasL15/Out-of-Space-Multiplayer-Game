package core

// TODO: REMOVE IF NEEDED.
import (
	"database/sql"
	"encoding/json"

	"github.com/go-sql-driver/mysql"
)

type JsonOut struct { // Structure for json output.
	Type string     `json:"type"` // See PROTOCOL.md
	Data []QueryOut `json:"data"`
} // Structure for the SQL output.
type QueryOut struct {
	MvpName  string `json:"mvpName"`
	MvpScore int    `json:"mvpScore"`
}

func getLeaderboard() string {
	// connect to root@localhost
	cfg := mysql.Config{
		User:                 "root",
		Passwd:               "",
		Net:                  "tcp",
		Addr:                 "127.0.0.1:3306",
		DBName:               "main-db",
		AllowNativePasswords: true, // Allows blank db password.
	}
	// Get a database handle and error handle.
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		panic(err)
	}
	result, err := db.Query("SELECT `mvpName`, " +
		"`mvpScore` FROM `gameLobbies` ORDER BY `mvpScore` DESC")
	if err != nil {
		panic(err)
	}
	// Get MVP name and score from table and order by score in decending order
	dataSlice := make([]QueryOut, 0, 49) // Make an array max 50 indexes.
	// this is using QueryOut[] structures.
	for result.Next() { // Iterating across each row
		var theTag QueryOut // Tags are used to validate if row is empty.
		err = result.Scan(&theTag.MvpName, &theTag.MvpScore)
		if err != nil {
			panic(err)
		}
		jsonRow := QueryOut{ // Define a new QueryOut{} struct.
			MvpName:  theTag.MvpName,
			MvpScore: theTag.MvpScore,
		}
		dataSlice = append(dataSlice, jsonRow)
		// Append it to the list of QueryOut{} structures.

		// Now assign the list of QueryOut{} structures to a JsonOut{}
		// structre.
	}
	requestOut := JsonOut{
		Type: "leaderboard_data",
		Data: dataSlice,
	} // Define a requestOut structure for the output

	mar, err := json.Marshal(requestOut)
	if err != nil {
		panic(err)
	}
	out := string(mar[:])
	return out // Final request, b is an array of bytes.
}
