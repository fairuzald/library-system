import postmanToOpenApi from "postman-to-openapi";

const postmanCollection = "./test/postman.json";
const outputFile = "./api-gateway/static/openapi.json";

async function convert() {
  try {
    await postmanToOpenApi(postmanCollection, outputFile, {
      defaultTag: "General",
      outputFormat: "json",
    });
  } catch (err) {
    console.error(err);
  }
}

convert();
