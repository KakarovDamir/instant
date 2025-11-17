# Project instant

One Paragraph of project description goes here

## Getting Started

Make sure that .env is configured, and after that run:
``` bash
docker compose up
```

Architecture:
![Architectire in c4](./architecture.svg "Instant")

> Alternatively you can look at `architecture.excalidraw` file

Description:



Demo of working kafka cluster:

<video width="320" height="240" controls>
  <source src="./instant_email_demo.mp4" type="video/mp4">
</video>


To get swagger documentation firstly run:
```bash
make swagger
```

Then open: http://localhost:8080/swagger


> Alternatively you can use Postman collection: `instant.postman_collection.json`


---
 Annotation: project was started using [go-blueprint](https://go-blueprint.dev/), so there are some artifacts in commit history.