export default function GridPage() {
  return (
    <div className="bg-black min-h-screen w-[800px] mt-[100px] mx-auto">
      <h3 className="text-white/80 text-xl mb-2">Mobile layout</h3>
      <div className="grid grid-cols-2 grid-rows-10 gap-3">
        <div className="1 bg-sky-700 row-span-10" />
        <div className="2 bg-sky-700 row-span-6" />
        <div className="3 bg-sky-700 row-span-10" />
        <div className="4 bg-sky-700 row-span-6" />
        <div className="5 bg-sky-700 row-span-10" />
        <div className="6 bg-sky-700 row-span-6" />
        <div className="7 bg-sky-700 row-span-10" />
        <div className="8 bg-sky-700 row-span-6" />
        <div className="9 bg-sky-700 row-span-10" />
        <div className="10 bg-sky-700 row-span-10" />
      </div>
      <div className="mt-[100px]" />
      <h3 className="text-white/80 text-xl mb-2">Tablet layout</h3>
      <div className="grid grid-cols-3 grid-rows-10 gap-3">
        <div className="1 text-white bg-sky-700 row-span-10">1</div>
        <div className="2 text-white bg-sky-700 row-span-6">2</div>
        <div className="3 text-white bg-sky-700 row-span-10">3</div>
        <div className="4 text-white bg-sky-700 row-span-6">4</div>
        <div className="5 text-white bg-sky-700 row-span-10">5</div>
        <div className="7 text-white bg-sky-700 row-span-10">7</div>
        <div className="6 text-white bg-sky-700 row-span-6">6</div>
        <div className="8 text-white bg-sky-700 row-span-6">8</div>
        <div className="9 text-white bg-sky-700 row-span-10">9</div>
        <div className="10 text-white bg-sky-700 row-span-10">10</div>
      </div>
      <div className="mt-[100px]" />
      <h3 className="text-white/80 text-xl mb-2">Desktop layout</h3>
      <div className="grid grid-cols-5 grid-rows-10 gap-3">
        <div className="1 bg-sky-700 row-span-10" />
        <div className="2 bg-sky-700 row-span-6" />
        <div className="3 bg-sky-700 row-span-10" />
        <div className="4 bg-sky-700 row-span-6" />
        <div className="5 bg-sky-700 row-span-10" />
        <div className="6 bg-sky-700 row-span-6" />
        <div className="8 bg-sky-700 row-span-6" />
        <div className="7 bg-sky-700 row-span-10" />
        <div className="9 bg-sky-700 row-span-10" />
        <div className="10 bg-sky-700 row-span-10" />
      </div>
      <div className="mt-[100px]" />
      <h3 className="text-white/80 text-xl mb-2">Responsive layout</h3>
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 grid-rows-10 gap-3 px-10">
        <div className="1 text-white bg-sky-700 row-span-10 order-1">1</div>
        <div className="2 text-white bg-sky-700 row-span-7 order-2">2</div>
        <div className="3 text-white bg-sky-700 row-span-10 order-3">3</div>
        <div className="4 text-white bg-sky-700 row-span-7 order-4">4</div>
        <div className="5 text-white bg-sky-700 row-span-10 order-5">5</div>

        <div className="6 text-white bg-sky-700 row-span-7 order-6 sm:order-7">6</div>
        <div className="7 text-white bg-sky-700 row-span-10 order-7 sm:order-6 lg:order-8">7</div>
        <div className="8 text-white bg-sky-700 row-span-7 order-8 lg:order-7">8</div>
        <div className="9 text-white bg-sky-700 row-span-10 order-9">9</div>
        <div className="10 text-white bg-sky-700 row-span-10 order-10">10</div>
      </div>
    </div>
  );
}
