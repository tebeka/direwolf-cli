Command line client for direwolf.

# Building

    cd dw && go build

# Testing

For testing you'll need to set `DW_API_KEY` environment variable.
Testing will run against `direwolf-brainard.herokuapp.com` direwolf on `brainard.herokudev.com` cloud.

    export DW_API_KEY=your-api-key
    cd dw && go test -v
