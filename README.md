# Out of Space
Our project is a game which is embedded in a web browser written in Godot with a Backend game server written in Go. 

## Requirements to run game locally
- Go 1.21.6 for running the server
- MySQL database

## File structure
### html
We've exported the game to HTML so that you can run the game locally but you need to change some HTML headers or it will not work. Please see [Godot's Documentation](https://docs.godotengine.org/en/stable/tutorials/export/exporting_for_web.html) to see what headers you need to change. 

The game will not work without the server so the server must be run using:
```
go run .
``` 
### frontend-src
The entire frontend part of our game written in Godot 4.

### backend-src
The Go backend for our server. This was setup to serve on our exported website so this will not work with the html file provied as it was setup to run on our game server.
### database
`db-schema.sql` is a dump of our `playtest` database without any data populated wheras `db-populated.sql` is our database with our playtesting data. `mysqldump` can be used to import this database to MySQL

Credits: Alex Gallon, David Wood, Junjia Liang, Ziyu Liao, Elias Liassides, Boey Jun Yang

