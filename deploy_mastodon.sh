#!/bin/bash

# Mastodon Deployment Script
SCRIPT_VERSION="v0.2.0"
MASTODON_VERSION="v4.2.10"

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check for required tools
for cmd in docker docker-compose curl certbot; do
    if ! command_exists $cmd; then
        echo "❌ $cmd is not installed. Please install it and run this script again."
        exit 1
    fi
done

# Function to generate a random string
generate_random_string() {
    openssl rand -base64 32 | tr -d /=+ | cut -c -"$1"
}

# Function to check DNS propagation
check_dns() {
    local domain="$1"
    local ip="$2"
    local dns_ip=$(dig +short $domain)

    if [ "$dns_ip" = "$ip" ]; then
        return 0
    else
        return 1
    fi
}

# Main script starts here
echo "🚀 Welcome to the Mastodon Deployment Script $SCRIPT_VERSION"
echo "This script will guide you through setting up a Mastodon instance."

# Gather necessary information
read -p "Enter your domain name (e.g., mastodon.example.com): " DOMAIN_NAME
read -p "Enter your server's public IP address: " IP_ADDRESS

# Check DNS setup
echo "Checking DNS setup..."
if check_dns "$DOMAIN_NAME" "$IP_ADDRESS"; then
    echo "✅ DNS is correctly set up."
else
    echo "❌ DNS is not set up correctly. Please ensure your domain points to your server's IP address."
    echo "You can check DNS propagation at https://www.whatsmydns.net/#A/$DOMAIN_NAME"
    read -p "Have you set up the DNS correctly now? (yes/no): " dns_setup
    if [[ $dns_setup != "yes" ]]; then
        echo "Please set up DNS and run this script again."
        exit 1
    fi
fi

# Set up SSL/TLS
echo "Setting up SSL/TLS certificate..."
sudo certbot certonly --standalone -d $DOMAIN_NAME

if [ $? -ne 0 ]; then
    echo "❌ Failed to obtain SSL/TLS certificate. Please ensure your domain is correctly set up and try again."
    exit 1
fi

# Generate random passwords
DB_PASSWORD=$(generate_random_string 32)
REDIS_PASSWORD=$(generate_random_string 32)

# Create .env.production file
cat << EOF > .env.production
# Federation
LOCAL_DOMAIN=$DOMAIN_NAME
SINGLE_USER_MODE=false
ENABLE_REGISTRATIONS=true

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=$REDIS_PASSWORD

# PostgreSQL
DB_HOST=db
DB_PORT=5432
DB_NAME=mastodon
DB_USER=mastodon
DB_PASS=$DB_PASSWORD

# Secrets (generated automatically)
SECRET_KEY_BASE=$(generate_random_string 128)
OTP_SECRET=$(generate_random_string 128)

# VAPID keys (generated automatically)
VAPID_PRIVATE_KEY=$(openssl ecparam -name prime256v1 -genkey -noout -out /dev/null 2>&1 | openssl ec -in /dev/stdin -outform DER 2>/dev/null | tail -c +8 | head -c 32 | base64)
VAPID_PUBLIC_KEY=$(echo -n "$VAPID_PRIVATE_KEY" | openssl ec -in /dev/stdin -inform DER -pubout -outform DER 2>/dev/null | tail -c 65 | base64)

# Sending mail (update with your SMTP details)
SMTP_SERVER=smtp.example.com
SMTP_PORT=587
SMTP_LOGIN=your_smtp_login
SMTP_PASSWORD=your_smtp_password
SMTP_FROM_ADDRESS=mastodon@$DOMAIN_NAME

# File storage (local)
PAPERCLIP_ROOT_PATH=/opt/mastodon/public/system
EOF

# Create docker-compose.yml file
cat << EOF > docker-compose.yml
version: '3'
services:
  db:
    image: postgres:14-alpine
    restart: always
    environment:
      - POSTGRES_USER=mastodon
      - POSTGRES_DB=mastodon
      - POSTGRES_PASSWORD=$DB_PASSWORD
    volumes:
      - ./postgres:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    restart: always
    command: redis-server --requirepass $REDIS_PASSWORD
    volumes:
      - ./redis:/data

  web:
    image: tootsuite/mastodon:$MASTODON_VERSION
    restart: always
    env_file: .env.production
    command: bash -c "rm -f /mastodon/tmp/pids/server.pid; bundle exec rails s -p 3000"
    ports:
      - "127.0.0.1:3000:3000"
    depends_on:
      - db
      - redis
    volumes:
      - ./public/system:/mastodon/public/system

  streaming:
    image: tootsuite/mastodon:$MASTODON_VERSION
    restart: always
    env_file: .env.production
    command: node ./streaming
    ports:
      - "127.0.0.1:4000:4000"
    depends_on:
      - db
      - redis

  sidekiq:
    image: tootsuite/mastodon:$MASTODON_VERSION
    restart: always
    env_file: .env.production
    command: bundle exec sidekiq
    depends_on:
      - db
      - redis
    volumes:
      - ./public/system:/mastodon/public/system
EOF

# Set up Nginx
echo "Setting up Nginx..."
sudo tee /etc/nginx/sites-available/mastodon << EOF
server {
    listen 80;
    listen [::]:80;
    server_name $DOMAIN_NAME;
    return 301 https://\$server_name\$request_uri;
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name $DOMAIN_NAME;

    ssl_certificate /etc/letsencrypt/live/$DOMAIN_NAME/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/$DOMAIN_NAME/privkey.pem;

    root /opt/mastodon/public;

    location / {
        try_files \$uri @proxy;
    }

    location @proxy {
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header Proxy "";
        proxy_pass_header Server;

        proxy_pass http://127.0.0.1:3000;
        proxy_buffering off;
        proxy_redirect off;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    location /api/v1/streaming {
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header Proxy "";

        proxy_pass http://127.0.0.1:4000;
        proxy_buffering off;
        proxy_redirect off;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    location /system {
        add_header Cache-Control "public, max-age=31536000, immutable";
        add_header Strict-Transport-Security "max-age=31536000";
    }

    error_page 500 501 502 503 504 /500.html;
}
EOF

sudo ln -s /etc/nginx/sites-available/mastodon /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx

# Start Docker containers
echo "Starting Docker containers..."
docker-compose up -d

# Create first admin user
echo "Creating first admin user..."
docker-compose run --rm web bash -c "RAILS_ENV=production bundle exec rails mastodon:make_admin USERNAME=admin EMAIL=admin@$DOMAIN_NAME"

# Final messages
echo "✅ Mastodon deployment completed successfully!"
echo "🌐 Your Mastodon instance is now available at https://$DOMAIN_NAME"
echo "👤 An admin user has been created with the following credentials:"
echo "   Username: admin"
echo "   Email: admin@$DOMAIN_NAME"
echo "⚠️  Please log in and change the admin password immediately!"
echo "📚 For more information on managing your Mastodon instance, visit: https://docs.joinmastodon.org/"