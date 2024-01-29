# syntax=docker/dockerfile:1

# Comments are provided throughout this file to help you get started.
# If you need more help, visit the Dockerfile reference guide at
# https://docs.docker.com/go/dockerfile-reference/

ARG NODE_VERSION=lts
ARG UNKEY_ROOT_KEY
ARG ARTILLERY_CLOUD_API_KEY
ARG BASE_URL=https://canary.unkey.dev

FROM node:${NODE_VERSION}-alpine




WORKDIR /usr/src/app

RUN npm i -g artillery

COPY . .


ENV UNKEY_ROOT_KEY $UNKEY_ROOT_KEY
ENV ARTILLERY_CLOUD_API_KEY $ARTILLERY_CLOUD_API_KEY
ENV FLY_REGION empty
# Run the application.
CMD ["artillery", "run", "./keys.verifyKey.yaml", "--record", "--tags platform:fly,region:${FLY_REGION}", "--target=${BASE_URL}", ]
