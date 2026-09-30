import { createApp } from './app.js';

const port = process.env.PORT || 3001;
createApp().listen(port, '0.0.0.0', () => {
  console.log(`node-api listening on 0.0.0.0:${port}`);
});
