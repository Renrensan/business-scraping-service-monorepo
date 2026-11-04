FROM node:20-alpine

WORKDIR /app

COPY package.json ./
COPY build ./build
COPY node_modules ./node_modules

EXPOSE 3000
ENV PORT=3000
ENV HOST=0.0.0.0
ENV NODE_ENV=production

CMD ["node", "build"]

