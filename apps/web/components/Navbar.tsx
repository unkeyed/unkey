"use client";

import Link from "next/link";

export const NavigationBar = () => {
  const showNavMobile = () => {
    const navMobile = document.getElementById("navbar-cta");
    navMobile?.classList.toggle("hidden");
  };
  return (
    <nav className="bg-gray-50 dark:bg-gray-900">
      <div className="flex flex-wrap items-center justify-between max-w-screen-xl p-4 mx-auto">
        <Link href="/" className="flex items-center">
          <span className="self-center text-2xl font-semibold whitespace-nowrap dark:text-white">
            Unkey
          </span>
        </Link>
        <div className="flex md:order-2">
          <Link
            href="/auth/sign-up"
            target="_blank"
            className="px-4 py-2 mr-3 text-sm font-medium text-center text-white bg-gray-800 rounded-lg hover:bg-gray-500 focus:ring-4 focus:outline-none focus:ring-gray-300 dark:bg-gray-600 dark:hover:bg-gray-500 dark:focus:ring-gray-800"
          >
            Dashboard
          </Link>
          <button
            onClick={showNavMobile}
            type="button"
            className="inline-flex items-center text-sm text-gray-500 rounded-lg md:hidden hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-200 dark:text-gray-400 dark:hover:bg-gray-700 dark:focus:ring-gray-600"
          >
            <span className="sr-only">Open main menu</span>
            <svg
              className="w-6 h-6"
              aria-hidden="true"
              fill="currentColor"
              viewBox="0 0 20 20"
              xmlns="http://www.w3.org/2000/svg"
            >
              <title>menu</title>
              <path
                fillRule="evenodd"
                d="M3 5a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zM3 10a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zM3 15a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        </div>
        <div
          className="items-center justify-between hidden w-full md:flex md:w-auto md:order-1"
          id="navbar-cta"
        >
          <ul className="flex flex-col p-4 mt-4 font-medium rounded-lg md:p-0 bg-gray-50 md:flex-row md:space-x-8 md:mt-0 md:bg-gray-50 dark:bg-gray-800 md:dark:bg-gray-900">
            <li>
              <Link
                href="/"
                className="block py-2 pl-3 pr-4 text-center text-whiterounded md:bg-transparent md:text-gray-800 md:p-0 md:dark:text-gray-500"
              >
                Home
              </Link>
            </li>
            <li>
              <Link
                href="/pricing"
                className="block py-2 pl-3 pr-4 text-center text-gray-900 rounded hover:bg-gray-100 md:hover:bg-transparent md:hover:text-gray-500 md:p-0 md:dark:hover:text-gray-500 dark:text-white dark:hover:bg-gray-700 dark:hover:text-white md:dark:hover:bg-transparent"
              >
                Pricing
              </Link>
            </li>
            <li>
              <Link
                href="/docs"
                className="block py-2 pl-3 pr-4 text-center text-gray-900 rounded hover:bg-gray-100 md:hover:bg-transparent md:hover:text-gray-500 md:p-0 md:dark:hover:text-gray-500 dark:text-white dark:hover:bg-gray-700 dark:hover:text-white md:dark:hover:bg-transparent"
              >
                Docs
              </Link>
            </li>
            <li>
              <Link
                href="/blog"
                className="block py-2 pl-3 pr-4 text-center text-gray-900 rounded hover:bg-gray-100 md:hover:bg-transparent md:hover:text-gray-500 md:p-0 md:dark:hover:text-gray-500 dark:text-white dark:hover:bg-gray-700 dark:hover:text-white md:dark:hover:bg-transparent "
              >
                Blog
              </Link>
            </li>
            <li>
              <Link
                href="/changelog"
                className="block py-2 pl-3 pr-4 text-center text-gray-900 rounded hover:bg-gray-100 md:hover:bg-transparent md:hover:text-gray-500 md:p-0 md:dark:hover:text-gray-500 dark:text-white dark:hover:bg-gray-700 dark:hover:text-white md:dark:hover:bg-transparent"
              >
                Changelog
              </Link>
            </li>
            <li>
              <Link
                href="/about"
                className="block py-2 pl-3 pr-4 text-center text-gray-900 rounded hover:bg-gray-100 md:hover:bg-transparent md:hover:text-gray-500 md:p-0 md:dark:hover:text-gray-500 dark:text-white dark:hover:bg-gray-700 dark:hover:text-white md:dark:hover:bg-transparent"
              >
                About
              </Link>
            </li>
            <li>
              <Link
                href="/discord"
                className="block py-2 pl-3 pr-4 text-center text-gray-900 rounded hover:bg-gray-100 md:hover:bg-transparent md:hover:text-gray-500 md:p-0 md:dark:hover:text-gray-500 dark:text-white dark:hover:bg-gray-700 dark:hover:text-white md:dark:hover:bg-transparent"
              >
                Discord
              </Link>
            </li>
          </ul>
        </div>
      </div>
    </nav>
  );
};
