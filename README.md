# idunion.web

Web interface of the IDUnion network for registering and managing entities.

## Architecture & Design

* created with htmx
	* allows thin clients and thus fast performance experiences
	* data handling happens mostly in the backend which is a benefit when it comes to processing secure data (i.e. FeatureFlags can solely handled in the backend)

## Build

```
docker build -t idunion.web .
```

## Run

```
docker run -p 3000:8080 idunion.web
```

* Open http://localhost:3000 in your browser
