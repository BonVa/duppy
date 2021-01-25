# duppy
## About
Duppy has a "dispatcher and worker" pattern (no disrespect, please think of it as a manager and some workers), which can be used in high-performance scenarios, like web API servers, consuming jobs... It may not be the fastest or the greatest M&W design, but it would be one of the simplest and easiest to understand and deploy in your real-life work.

## Usage
``` go
import "github.com/BonVa/duppy"

func main() {
  dMaster := Duppy.Master(config) //a map, or just put a nil map, it will create an default master who has 10 workers
  
  // as a web server r as a consumer
  dMaster.Listen(jobSocket) // from MQ, channel or raw network socket
}
```
