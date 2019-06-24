# copy composer files from composer image
FROM composer:latest AS composer
FROM alpine:3.9
COPY --from=composer /usr/bin/composer /usr/bin/composer

RUN ln -s /var/www/html ${SITE_WORKDIR_IN_CONTAINER} \
    && apk --no-cache add php7 php7-fpm php7-mysqli php7-json php7-openssl php7-curl \
           php7-zlib php7-xml php7-phar php7-intl php7-dom php7-xmlreader php7-ctype \
           php7-mbstring php7-gd php7-xdebug supervisor curl git

WORKDIR ${SITE_WORKDIR_IN_CONTAINER}

ENV COMPOSER_ALLOW_SUPERUSER 1

CMD ["php-fpm"]

# listen 9000 port
EXPOSE 9000