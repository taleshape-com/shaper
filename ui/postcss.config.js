import tailwindcss from 'tailwindcss';
import autoprefixer from 'autoprefixer';

export default {
  plugins: [
    tailwindcss({ config: 'ui/tailwind.config.js' }),
    autoprefixer,
  ],
}
