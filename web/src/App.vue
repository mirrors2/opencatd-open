<template>
  <div class="flex flex-col h-screen w-screen bg-diy">
    <header class="bg-zinc-700 text-white py-4 fixed top-0 w-full z-10">
      <div class="container  flex justify-between items-center px-4">
        <div class="flex items-center">
          <img class="h-8" src="./assets/logo.svg" alt="Logo">
          <h1 class="ml-2 text-l">opencatd-open</h1>
        </div>
        <div class="flex justify-between items-center space-x-2">
          <a href="https://github.com/mirrors2/opencatd-open">
            <img class="h-6" alt="GitHub Repo stars"
              src="https://img.shields.io/github/stars/mirrors2/opencatd-open?style=social">
          </a>
          <!-- <button class="bg-white text-gray-900 font-medium py-2 px-4 rounded-full hover:bg-gray-200 hover:text-gray-900">Log In</button> -->
        </div>
      </div>
    </header>

    <main class="w-screen  flex-grow flex flex-col justify-center items-center bg-zinc-700 mt-16">
      <div class="">
        <div class=" flex items-center justify-center my-0">
          <img src="./assets/logo.svg" alt="Project Logo" class="logo h-10 my-0">
        </div>

        <div class=" flex items-center justify-center my-0">
          <a class="text-gray-300 text-4xl" href="https://github.com/mirrors2/opencatd-open">opencatd-open</a>
        </div>

        <div class="my-12">
          <p class="text-gray-300 mt-2 mx-5">opencatd-open is an open-source, team-shared service for ChatGPT API that can
            be safely shared with others for API usage.</p>
        </div>
      </div>
      <div class="mt-8 shadow-lg mb-8 flax justify-center items-center">
        <div class="flex gap-2">
          <div>
            <input class="w-60 py-2 px-2 bg-zinc-700 rounded-lg border focus:outline-none text-white flex-1" disabled name=""
              id="" v-model="url">
          </div>
          <br>
          <div>
            <input type="submit" value="复制" @click="copyUrl"
              class="bg-white font-medium py-2 px-3 hover:bg-gray-200 hover:text-gray-900 rounded-lg h-10 text-gray-700">
          </div>
        </div>
      </div>

      <div class="bg-diy w-screen h-auto flex flex-col overflow-x-auto  justify-center items-center">
        <div>
          <p class="text-gray-500 mt-5 text-sm">👉Api-Keys: <a
              href=https://platform.openai.com/account/api-keys>https://platform.openai.com/account/api-keys</a></p>
        </div>
        <section
          class="w-10/12 h-auto mt-5 my-4 shadow-lg mb-8 p-12 bg-white border-2 border-gray-200 rounded-xl hover:shadow-2xl">
          <!-- card content -->
          <div class="flex justify-center">
            <h1 class="text-4xl my-2">使用说明</h1>
          </div>
          <hr class="my-5">
          <div class="text-xl mt-5">
            <h2>作为OpenAI API代理</h2>
          </div>
          <div class="my-4 gap">
            <p>由于OpenAI API不能再国内访问，使用"openai api key+自定义域名"可以无感访问</p>
            <p>在自定义地址中填入当前地址</p> <img class=" sm:max-w-full md:max-w-sm h-auto" src="./assets/usersdomain.jpg" alt="">
          </div>
        </section>
      </div>
      <div class="bg-diy w-screen h-auto flex flex-grow overflow-x-auto  justify-center">
        <section
          class="w-10/12 h-auto my-2 shadow-lg mb-8 p-12 bg-white border-2 border-gray-200 rounded-xl hover:shadow-2xl">
          <!-- card content -->
          <div class="text-xl">
            <h2>团队共享API模式</h2>
          </div>
          <div class="my-4 gap">
            <p>团队共享模式可以把openai api key分发给多人使用.使用openai api key作为内部访问密钥</p>
            <p>系统生成api-key,使用"系统生成的api-key+自定义域名"可以无感访问</p>
            <p></p>
            <p>在自定义地址中填入当前地址。以<span class="text-rose-500">OpenCat</span>为例(目前体验最好):</p> <img
              class=" sm:max-w-full md:max-w-sm h-auto" src="./assets/team.jpg" alt="">
            <hr class="my-5">
            <blockquote>
              <p>注意:第三方应用需要支持自定义 OpenAI Key 和 Host</p>
            </blockquote>

          </div>
        </section>
      </div>
    </main>

    <footer class="bg-diy py-6 w-screen flex flex-col justify-center items-center">
      <div class="bg-diy w-10/12 h-auto flex overflow-x-auto  justify-between mx-60">
        <ul class="flex space-x-4">
          <li><a href="https://github.com/mirrors2/opencatd-open#qa" class="text-gray-700 hover:text-gray-900">FAQ</a>
          </li>
        </ul>
        <p class="text-gray-700">© {{ currentYear }} <a href="https://github/mirrors2/opencatd-open">Sakurasan</a>. All
          Rights
          Reserved.</p>
      </div>
    </footer>

  </div>
</template>
  
<script setup>
import { ref, reactive, onMounted } from 'vue';
import HelloWorld from './components/HelloWorld.vue'

const currentYear = ref('');
let url = ref('')

const copyUrl = () => {
  navigator.clipboard.writeText(url.value).then(() => {
    alert('复制成功！')
  }, () => {
    alert('复制失败，请手动复制。')
  })

}
const getcurrentYear = () => {
  currentYear.value = new Date().getFullYear().toString()
}



onMounted(() => {
  url.value = window.location.origin;
  getcurrentYear();
})


</script>
  
<style scoped>
.bg-diy {
  background-color: #f0f0f0;
}

.logo {
  height: 6em;
  padding: 1.5em;
  will-change: filter;
  transition: filter 300ms;

}

.logo:hover {
  filter: drop-shadow(0 0 3em #45f5e3aa);
}

p {
  margin-bottom: 4px;
}

blockquote {
  padding: 0 1em;
  border-left: 0.25em solid #838989aa;
}
</style>