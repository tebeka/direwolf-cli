Command line client for direwolf.

# Building

    cd dw && go build

# Testing

For testing you'll need to set `DW_API_KEY` environment variable.

    export DW_API_KEY=your-api-key
    cd dw && go test -v
