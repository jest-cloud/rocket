package rocket

import (
	"html/template"
	"net/http"
)

// PlaygroundType determines which playground interface to use
type PlaygroundType string

const (
	PlaygroundTypeApolloSandbox PlaygroundType = "apollo"     // Apollo Sandbox (recommended)
	PlaygroundTypeGraphiQL      PlaygroundType = "graphiql"   // GraphiQL (stable)
	PlaygroundTypePlayground    PlaygroundType = "playground" // GraphQL Playground (legacy)
)

// PlaygroundHandler creates an HTTP handler that serves a GraphQL playground
// Uses Apollo Sandbox by default (most modern and stable)
func PlaygroundHandler(endpoint string) http.HandlerFunc {
	return PlaygroundHandlerWithType(endpoint, PlaygroundTypeApolloSandbox)
}

// PlaygroundHandlerWithType creates a playground handler with a specific type
func PlaygroundHandlerWithType(endpoint string, playgroundType PlaygroundType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only serve playground on GET requests
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check if this is a GraphQL query request
		if r.URL.Query().Get("query") != "" {
			// This is an actual query, not a playground request
			// Let the main handler deal with it
			http.Error(w, "Use POST for GraphQL queries", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		
		var htmlTemplate string
		switch playgroundType {
		case PlaygroundTypeApolloSandbox:
			htmlTemplate = apolloSandboxHTML
		case PlaygroundTypeGraphiQL:
			htmlTemplate = graphiqlHTML
		case PlaygroundTypePlayground:
			htmlTemplate = playgroundHTML
		default:
			htmlTemplate = apolloSandboxHTML
		}
		
		tmpl := template.Must(template.New("playground").Parse(htmlTemplate))
		tmpl.Execute(w, map[string]string{
			"Endpoint": endpoint,
		})
	}
}

// Apollo Sandbox (modern, recommended - Apollo's official playground)
const apolloSandboxHTML = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>GraphQL Playground - Apollo Sandbox</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		body {
			margin: 0;
			overflow: hidden;
		}
		#embedded-sandbox {
			position: absolute;
			width: 100%;
			height: 100%;
		}
	</style>
</head>
<body>
	<div id="embedded-sandbox"></div>
	<script src="https://embeddable-sandbox.cdn.apollographql.com/_latest/embeddable-sandbox.umd.production.min.js"></script>
	<script>
		new window.EmbeddedSandbox({
			target: '#embedded-sandbox',
			initialEndpoint: '{{.Endpoint}}',
			includeCookies: true,
		});
	</script>
</body>
</html>`

// GraphiQL playground (more stable, better autocomplete)
const graphiqlHTML = `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>GraphQL Playground - GraphiQL</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		body {
			margin: 0;
			overflow: hidden;
		}
		#graphiql {
			height: 100vh;
		}
	</style>
	<link href="https://unpkg.com/graphiql@3.0.10/graphiql.min.css" rel="stylesheet" />
</head>
<body>
	<div id="graphiql">Loading...</div>
	<script
		crossorigin
		src="https://unpkg.com/react@18/umd/react.production.min.js"
	></script>
	<script
		crossorigin
		src="https://unpkg.com/react-dom@18/umd/react-dom.production.min.js"
	></script>
	<script
		crossorigin
		src="https://unpkg.com/graphiql@3.0.10/graphiql.min.js"
	></script>
	<script>
		const fetcher = GraphiQL.createFetcher({
			url: '{{.Endpoint}}',
		});

		const root = ReactDOM.createRoot(document.getElementById('graphiql'));
		root.render(
			React.createElement(GraphiQL, {
				fetcher: fetcher,
				defaultEditorToolsVisibility: true,
			})
		);
	</script>
</body>
</html>`

// GraphQL Playground (original)
const playgroundHTML = `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>GraphQL Playground</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react@1.7.28/build/static/css/index.css">
	<link rel="shortcut icon" href="https://cdn.jsdelivr.net/npm/graphql-playground-react@1.7.28/build/favicon.png">
	<script src="https://cdn.jsdelivr.net/npm/graphql-playground-react@1.7.28/build/static/js/middleware.js"></script>
</head>
<body>
	<div id="root"></div>
	<script>
		window.addEventListener('load', function (event) {
			GraphQLPlayground.init(document.getElementById('root'), {
				endpoint: '{{.Endpoint}}',
				settings: {
					'request.credentials': 'same-origin',
				},
			})
		})
	</script>
</body>
</html>`

