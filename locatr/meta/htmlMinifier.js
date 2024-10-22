/**
 * List of reserved attributes to keep.
 * @type {string[]}
 */
const RESERVED_ATTRIBUTES = [
  "accept",
  "alt",
  "aria-checked",
  "aria-current",
  "aria-label",
  "aria-required",
  "aria-role",
  "aria-selected",
  "checked",
  "for",
  "href",
  "maxlength",
  "name",
  "pattern",
  "placeholder",
  "readonly",
  "required",
  "selected",
  "src",
  "text-value",
  "title",
  "type",
  "role",
  "value",
  "facet-refined", // Custom attribute for marking input-checkboxes of car-listing sites.
];

/**
 * Generates a hash code for a given string.
 * @param {string} str - The string to hash.
 * @returns {string} Hash code.
 */
function hashCode(str) {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = (hash << 5) - hash + char;
    hash |= 0; // Convert to 32bit integer
  }
  return hash.toString(36);
}

/**
 * Generates a unique ID for an element based on its locator.
 * @param {string} locator - The locator for the element.
 * @returns {string} Unique ID.
 */
function generateUniqueId(locator) {
  return hashCode(locator);
}

/**
 * Checks if an element is visible in the DOM.
 * @param {Element} element - The element to check.
 * @returns {boolean} True if the element is visible, false otherwise.
 */
function isElementVisible(element) {
  const style = window.getComputedStyle(element);
  const rect = element.getBoundingClientRect();
  const isVisible = !(
    element.getAttribute("aria-hidden") === "true" ||
    style.display === "none" ||
    style.visibility === "hidden" ||
    style.opacity === "0" ||
    rect.width === 0 ||
    rect.height === 0
  );
  return isVisible || Array.from(element.children).some(isElementVisible);
}

/**
 * Checks if an element is interactable.
 * @param {HTMLInputElement} element - The input element to check.
 * @returns {boolean} True if the element is interactable, false otherwise.
 */
function isInputInteractable(element) {
  const elemTagName = element.tagName.toLowerCase();
  if (elemTagName !== "input") return false;

  const elemType = (element.getAttribute("type") ?? "text")
    .toLowerCase()
    .trim();

  const clickableElemTypes = [
    "button",
    "checkbox",
    "date",
    "datetime-local",
    "email",
    "file",
    "image",
    "month",
    "number",
    "password",
    "radio",
    "range",
    "reset",
    "search",
    "submit",
    "tel",
    "text",
    "time",
    "url",
    "week",
  ];
  return clickableElemTypes.includes(elemType);
}

/**
 * Checks if an element is valid for processing.
 * @param {Element} element - The element to check.
 * @returns {boolean} True if the element is valid, false otherwise.
 */
function isValidElement(element) {
  const elemTagName = element.tagName.toLowerCase();
  if (elemTagName === "input") return isInputInteractable(element);

  const invalidTags = [
    "script",
    "style",
    "link",
    "iframe",
    "meta",
    "noscript",
    "path",
  ];
  return (
    !invalidTags.includes(elemTagName) &&
    element.getAttribute("aria-disabled") !== "true" &&
    !element.disabled &&
    isElementVisible(element)
  );
}

/**
 * Extracts visible text from an element without including text from child elements.
 * @param {Node} node - The node to extract text from.
 * @returns {string} Visible text within the current parent element.
 */
function getVisibleText(node) {
  if (node.nodeType === Node.ELEMENT_NODE && isElementVisible(node)) {
    let visibleText = "";
    for (const child of node.childNodes) {
      if (child.nodeType === Node.TEXT_NODE) {
        const trimmedText = child.data.trim();
        if (trimmedText) {
          visibleText += trimmedText + " ";
        }
      }
    }
    return visibleText.trim();
  }
  return "";
}

/**
 * @typedef {Object} Attributes
 * @property {string} [accept]
 * @property {string} [alt]
 * @property {string} [aria-checked]
 * @property {string} [aria-current]
 * @property {string} [aria-label]
 * @property {string} [aria-required]
 * @property {string} [aria-role]
 * @property {string} [aria-selected]
 * @property {string} [checked]
 * @property {string} [for]
 * @property {string} [href]
 * @property {string} [maxlength]
 * @property {string} [name]
 * @property {string} [pattern]
 * @property {string} [placeholder]
 * @property {string} [readonly]
 * @property {string} [required]
 * @property {string} [selected]
 * @property {string} [src]
 * @property {string} [text-value]
 * @property {string} [title]
 * @property {string} [type]
 * @property {string} [value]
 * @property {string} [role]
 * @property {string} [facet-refined]
 * @property {string} [data-*]
 */

/**
 * Extracts and trims attributes from an element.
 * @param {Element} element - The element to extract attributes from.
 * @returns {Attributes} Trimmed attributes.
 */
function getTrimmedAttributes(element) {
  /** @type {Attributes} */
  const trimmedAttributes = {};
  Array.from(element.attributes).forEach((attr) => {
    if (
      (RESERVED_ATTRIBUTES.includes(attr.name) ||
        attr.name.startsWith("data-")) &&
      attr.value !== ""
    ) {
      trimmedAttributes[attr.name] = attr.value;
    } else if (attr.name == "id" && attr.value != "") {
      trimmedAttributes["data-id"] = attr.value;
    }
  });
  return trimmedAttributes;
}

/**
 * Generates multiple CSS selectors for an element.
 * @param {Element} element - The element to generate CSS selectors for.
 * @returns {string[]} Array of CSS selectors.
 */
function generateCssSelectors(element) {
  /**
   * Escapes and joins class names.
   * @param {Element} el - The element whose class names to escape and join.
   * @returns {string} The escaped and joined class names.
   */
  function getClassNames(el) {
    if (typeof el.className === "string") {
      return el.className
        .split(/\s+/)
        .filter(Boolean)
        .map((cls) => `.${CSS.escape(cls)}`)
        .join("");
    }
    return "";
  }

  /**
   * Generates a CSS selector for an element using its ID.
   * @param {Element} el - The element to generate a CSS selector for.
   * @returns {string} The generated CSS selector.
   */
  function getIdSelector(el) {
    const path = [];
    while (el && el.nodeType === Node.ELEMENT_NODE) {
      let selector = el.nodeName.toLowerCase();
      if (el.id) {
        selector += `#${CSS.escape(String(el.id))}`;
        path.unshift(selector);
        break;
      }
      path.unshift(selector);
      el = el.parentNode;
    }
    return path.join(" > ");
  }

  /**
   * Generates a CSS selector for an element using its classes.
   * @param {Element} el - The element to generate a CSS selector for.
   * @returns {string} The generated CSS selector.
   */
  function getClassSelector(el) {
    const path = [];
    while (el && el.nodeType === Node.ELEMENT_NODE) {
      let selector = el.nodeName.toLowerCase();
      selector += getClassNames(el);
      path.unshift(selector);
      el = el.parentNode;
    }
    return path.join(" > ");
  }

  /**
   * Generates a CSS selector for an element using its type and :nth-of-type.
   * @param {Element} el - The element to generate a CSS selector for.
   * @returns {string} The generated CSS selector.
   */
  function getNthOfTypeSelector(el) {
    const path = [];
    while (el && el.nodeType === Node.ELEMENT_NODE) {
      let selector = el.nodeName.toLowerCase();
      const nth = getNthOfType(el);
      if (nth !== 1) {
        selector += `:nth-of-type(${nth})`;
      }
      path.unshift(selector);
      el = el.parentNode;
    }
    return path.join(" > ");
  }

  /**
   * Calculates the :nth-of-type index for an element.
   * @param {Element} el - The element to calculate the index for.
   * @returns {number} The :nth-of-type index.
   */
  function getNthOfType(el) {
    let nth = 1;
    let sibling = el;
    while (sibling.previousElementSibling) {
      sibling = sibling.previousElementSibling;
      if (sibling.nodeName.toLowerCase() === el.nodeName.toLowerCase()) {
        nth++;
      }
    }
    return nth;
  }

  return [
    getIdSelector(element),
    getNthOfTypeSelector(element),
    getClassSelector(element),
  ];
}

/**
 * @typedef {Object} ElementSpec
 * @property {string} id
 * @property {Attributes} attributes
 * @property {string} tag_name
 * @property {string} text
 * @property {ElementSpec[]} children
 */

/**
 * Creates an element specification object for an element.
 * @param {Element} element - The element to create a spec for.
 * @returns {ElementSpec|null} The element specification or null if invalid.
 */
function createElementSpec(element) {
  if (!isValidElement(element)) return null;

  const attrs = getTrimmedAttributes(element);
  const text = getVisibleText(element);

  const cssSelectors = generateCssSelectors(element);
  const uniqueId = generateUniqueId(cssSelectors[0]);

  const children = Array.from(element.children || [])
    .map(createElementSpec)
    .filter((el) => el !== null);

  return {
    tag_name: element.tagName,
    id: uniqueId,
    attributes: attrs,
    text,
    children,
  };
}

/**
 * Builds a tree structure of element specifications starting from a root element.
 * @param {Element} element - The root element to build the tree from.
 * @returns {ElementSpec|null} The element specification tree or null if invalid.
 */
function buildElementTree(element) {
  let elementSpec = createElementSpec(element);
  if (!elementSpec) return null;

  return elementSpec;
}

/**
 * Minifies the HTML document and returns a JSON string representation.
 * @returns {string} JSON string representation of the minified HTML.
 */
function minifyHTML() {
  const root = document.documentElement || document.body;
  return JSON.stringify(buildElementTree(root));
}

/**
 * Maps each element in the document to its CSS locator and unique ID.
 * @returns {string} JSON map of CSS locators to unique IDs.
 */
function getElementIdLocatorMap() {
  const elements = document.querySelectorAll("*");
  const map = {};

  elements.forEach((element) => {
    const cssSelectors = generateCssSelectors(element);
    const uniqueId = generateUniqueId(cssSelectors[0]);
    map[uniqueId] = cssSelectors;
  });

  return JSON.stringify(map, null, 2);
}

window.minifyHTML = minifyHTML;
window.mapElementsToJson = mapElementsToJson;
