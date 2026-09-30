import { createApp } from './app.js';

const port = process.env.PORT || 3001;
createApp().listen(port, () => {
  console.log(`node-api listening on port ${port}`);
});
