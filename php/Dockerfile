FROM php:7.3-fpm

ENV NEEDED_PHP_MODULES xdebug mysqli pdo_mysql soap zip gd
RUN ln -s /var/www/html /code

WORKDIR /code
#add php extensions installer.
ADD https://raw.githubusercontent.com/mlocati/docker-php-extension-installer/master/install-php-extensions /usr/local/bin/

RUN apt-get update && apt-get install -y wget nano git



#INSTALL PHP MODULES:
#see available PHP modules - https://github.com/mlocati/docker-php-extension-installer

RUN chmod uga+x /usr/local/bin/install-php-extensions && sync && \
    install-php-extensions $NEEDED_PHP_MODULES
    

## Install mcrypt
#RUN apt-get install -y libmcrypt-dev
#RUN docker-php-ext-install mcrypt


# INSTALL COMPOSER
RUN wget https://getcomposer.org/installer -O - -q \
    | php -- --install-dir=/bin --filename=composer --quiet
RUN composer --version

RUN usermod -u 1000 www-data

CMD ["php-fpm"]

# listen 9000 port
EXPOSE 9000

RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
