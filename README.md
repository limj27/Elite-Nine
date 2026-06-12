# Elite Nine

## Client
- Simple web UI for now but want to add mobile functionality (iOS & Android)

## Backend
- Websocket REST API Golang server to handle real-time gameplay
- Using Docker for ease of deployment on a Raspberry Pi
- Database to store trivia questions
- Using Redis to store session cache

## Things to Implement:
- Grid based on user's favorite team
In terms of difficulty, the creator picks the diffculty of the room and I think it would be nice to show that information on the list of rooms next to the status of the room. If both players have a different favorite team then both of them would be included in the grid (I am thinking maybe we can search through grids depending on the favorite teams), if it overlaps then only one team would be included. On easy mode, it would go Row 1: user 1's favorite team, Row 2 & 3: stat based and then Col 1: user2's favorite team (random one if they have the same favorite team) and then for Col 2 & 3 stat based. This would mean that we would have to have enough grids to satisfy all combinations of favorite teams. For username change it should check for duplicates (whether if it is available), For deleting account it shouldn't delete game history  for those who played against that person. Password change would be good to include, maybe we can include game history in that page as well for now since we already track it. I think it would be nice to have the setting accessible from the lobby topbar with a gear icon.
- Change the color of ready button
- Turn Timer (Feature to select when making the room)
