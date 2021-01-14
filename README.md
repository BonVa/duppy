# duppy
## About
Duppy has a "master and worker" pattern, which can be used in high performence scenarios, like web api server, consuming jobs...
It may not be the fastest or greatest M&W design, but it would be one of the simplest and eastiest to understand and deploy in your real-life work.

## Usage
``` go
import "github.com/BonVa/duppy"

func main() {
  dMaster := Duppy.Master(config) //a map, or just put a nil map, it will create an default master who has 10 workers
  
  // as a web server r as a consumer
  dMaster.Listen(jobSocket) // from MQ, channel or raw network socket
}
```
