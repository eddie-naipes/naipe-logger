FROM ruby:3.2-slim

RUN apt-get update && apt-get install -y build-essential && \
    gem install fpm

WORKDIR /AppSettings